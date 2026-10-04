//go:build !js

package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/adsouza/africa2ice/internal/application"
)

var errCrash = errors.New("injected crash")

// crashingFileSystem is the real disk until step failAt, where it simulates
// the process dying: that step fails, and every later call fails too. A torn
// crash on a Write first writes half the bytes, as a power loss mid-write
// would.
type crashingFileSystem struct {
	osFileSystem
	failAt  int
	torn    bool
	steps   int
	crashed bool
}

// step counts one mutating operation and reports whether it is the crash.
func (fs *crashingFileSystem) step() error {
	if fs.crashed {
		return errCrash
	}
	fs.steps++
	if fs.steps == fs.failAt {
		fs.crashed = true
		return errCrash
	}
	return nil
}

func (fs *crashingFileSystem) ReadFile(path string) ([]byte, error) {
	if fs.crashed {
		return nil, errCrash
	}
	return fs.osFileSystem.ReadFile(path)
}

func (fs *crashingFileSystem) Glob(pattern string) ([]string, error) {
	if fs.crashed {
		return nil, errCrash
	}
	return fs.osFileSystem.Glob(pattern)
}

func (fs *crashingFileSystem) CreateTemp(directory, pattern string) (temporaryFile, error) {
	if err := fs.step(); err != nil {
		return nil, err
	}
	file, err := fs.osFileSystem.CreateTemp(directory, pattern)
	if err != nil {
		return nil, err
	}
	return &crashingTemporary{temporaryFile: file, fs: fs}, nil
}

func (fs *crashingFileSystem) Rename(from, to string) error {
	if err := fs.step(); err != nil {
		return err
	}
	return fs.osFileSystem.Rename(from, to)
}

func (fs *crashingFileSystem) Remove(path string) error {
	if err := fs.step(); err != nil {
		return err
	}
	return fs.osFileSystem.Remove(path)
}

func (fs *crashingFileSystem) SyncDir(directory string) error {
	if err := fs.step(); err != nil {
		return err
	}
	return fs.osFileSystem.SyncDir(directory)
}

type crashingTemporary struct {
	temporaryFile
	fs *crashingFileSystem
}

func (file *crashingTemporary) Write(data []byte) (int, error) {
	if err := file.fs.step(); err != nil {
		if file.fs.torn && file.fs.steps == file.fs.failAt {
			written, _ := file.temporaryFile.Write(data[:len(data)/2])
			return written, err
		}
		return 0, err
	}
	return file.temporaryFile.Write(data)
}

func (file *crashingTemporary) Sync() error {
	if err := file.fs.step(); err != nil {
		return err
	}
	return file.temporaryFile.Sync()
}

func (file *crashingTemporary) Close() error {
	// A dead process's descriptors are closed by the OS, so the real file is
	// always closed; only the reported outcome reflects the crash.
	closeErr := file.temporaryFile.Close()
	if err := file.fs.step(); err != nil {
		return err
	}
	return closeErr
}

// slotOutcome is what a freshly started process sees in one slot.
type slotOutcome struct {
	turn    int
	deleted bool
}

func recoverSlot(t *testing.T, directory string, slot application.SlotID) slotOutcome {
	t.Helper()
	repository, err := NewFileRepository(directory, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repository.Close() }()
	if err := repository.BeginRead(1, slot); err != nil {
		t.Fatal(err)
	}
	read := waitCompletion(t, repository)
	if errors.Is(read.Err, os.ErrNotExist) {
		return slotOutcome{deleted: true}
	}
	if read.Err != nil || read.State == nil {
		t.Fatalf("recovered slot is neither a complete save nor a deletion: %v", read.Err)
	}
	return slotOutcome{turn: read.State.Turn}
}

func savedState(t *testing.T, turn int) application.SaveState {
	t.Helper()
	service, err := application.NewGameService(42)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	state.Turn = turn
	return state
}

// DESIGN.md §9: failure injection stops after every file-protocol step; the
// slot must then resolve to the complete old save, the complete new save, or a
// committed deletion, never a mixture. Each case starts from a committed save,
// runs one operation on a disk that crashes at step k, restarts on a healthy
// disk, and reads the slot. k grows until the operation completes with no
// crash, so every step is covered, including directory syncs and the cleanup
// after the commit.
func TestFileProtocolSurvivesACrashAtEveryStep(t *testing.T) {
	const slot = application.QuickSave
	operations := map[string]struct {
		run  func(*FileRepository) error
		next slotOutcome
	}{
		"overwrite": {run: func(repository *FileRepository) error {
			return repository.BeginWrite(2, slot, savedState(t, 8))
		}, next: slotOutcome{turn: 8}},
		"delete": {run: func(repository *FileRepository) error {
			return repository.BeginDelete(2, slot)
		}, next: slotOutcome{deleted: true}},
	}
	previous := slotOutcome{turn: 7}
	for name, operation := range operations {
		for _, torn := range []bool{false, true} {
			sawBoth := map[bool]bool{}
			for failAt := 1; ; failAt++ {
				directory := t.TempDir()
				baseline, err := NewFileRepository(directory, nil)
				if err != nil {
					t.Fatal(err)
				}
				if err := baseline.BeginWrite(1, slot, savedState(t, 7)); err != nil {
					t.Fatal(err)
				}
				if completion := waitCompletion(t, baseline); completion.Err != nil {
					t.Fatal(completion.Err)
				}
				_ = baseline.Close()

				fs := &crashingFileSystem{failAt: failAt, torn: torn}
				repository, err := newFileRepository(directory, nil, fs)
				if err != nil {
					t.Fatal(err)
				}
				if err := operation.run(repository); err != nil {
					t.Fatal(err)
				}
				completion := waitCompletion(t, repository)
				_ = repository.Close()

				got := recoverSlot(t, directory, slot)
				if got != previous && got != operation.next {
					t.Fatalf("%s, torn %t, crash at step %d: recovered %+v, want %+v or %+v",
						name, torn, failAt, got, previous, operation.next)
				}
				if completion.Err == nil && got != operation.next {
					t.Fatalf("%s, crash at step %d: reported success but recovered %+v", name, failAt, got)
				}
				sawBoth[got == operation.next] = true
				if leftovers, _ := filepath.Glob(filepath.Join(directory, temporaryPattern)); len(leftovers) != 0 {
					t.Fatalf("%s, crash at step %d: restart left %d temporary records", name, failAt, len(leftovers))
				}
				if !fs.crashed {
					break // the operation ran to completion: every step is covered
				}
			}
			if !sawBoth[true] || !sawBoth[false] {
				t.Fatalf("%s, torn %t: crashes only ever recovered one outcome (%v); the sweep missed the commit point", name, torn, sawBoth)
			}
		}
	}
}

// recordingFileSystem logs every mutating call the protocol makes.
type recordingFileSystem struct {
	osFileSystem
	calls []string
}

func (fs *recordingFileSystem) CreateTemp(directory, pattern string) (temporaryFile, error) {
	fs.calls = append(fs.calls, "create")
	file, err := fs.osFileSystem.CreateTemp(directory, pattern)
	if err != nil {
		return nil, err
	}
	return &recordingTemporary{temporaryFile: file, fs: fs}, nil
}

func (fs *recordingFileSystem) Rename(from, to string) error {
	fs.calls = append(fs.calls, "rename "+filepath.Base(to))
	return fs.osFileSystem.Rename(from, to)
}

func (fs *recordingFileSystem) SyncDir(directory string) error {
	fs.calls = append(fs.calls, "syncdir")
	return fs.osFileSystem.SyncDir(directory)
}

func (fs *recordingFileSystem) Remove(path string) error {
	fs.calls = append(fs.calls, "remove "+filepath.Base(path))
	return fs.osFileSystem.Remove(path)
}

type recordingTemporary struct {
	temporaryFile
	fs *recordingFileSystem
}

func (file *recordingTemporary) Write(data []byte) (int, error) {
	file.fs.calls = append(file.fs.calls, "write")
	return file.temporaryFile.Write(data)
}

func (file *recordingTemporary) Sync() error {
	file.fs.calls = append(file.fs.calls, "sync")
	return file.temporaryFile.Sync()
}

func (file *recordingTemporary) Close() error {
	file.fs.calls = append(file.fs.calls, "close")
	return file.temporaryFile.Close()
}

// The crash sweep cannot observe durability: on a healthy disk a missing
// directory sync changes nothing visible. This pins the order instead. Each
// record is written, synced, closed, renamed, and then its directory synced,
// and the world record is published before the metadata that commits it.
func TestFileProtocolPublishesEachRecordDurablyInOrder(t *testing.T) {
	directory := t.TempDir()
	fs := &recordingFileSystem{}
	repository, err := newFileRepository(directory, nil, fs)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repository.Close() }()
	if err := repository.BeginWrite(1, application.Manual1, savedState(t, 3)); err != nil {
		t.Fatal(err)
	}
	if completion := waitCompletion(t, repository); completion.Err != nil {
		t.Fatal(completion.Err)
	}
	var published []string
	for index, call := range fs.calls {
		if len(call) < len("rename ") || call[:len("rename ")] != "rename " {
			continue
		}
		published = append(published, call[len("rename "):])
		if index < 4 || fs.calls[index-4] != "create" || fs.calls[index-3] != "write" || fs.calls[index-2] != "sync" || fs.calls[index-1] != "close" {
			t.Fatalf("%s was not created, written, synced, and closed before its rename: %v", call, fs.calls)
		}
		if index+1 >= len(fs.calls) || fs.calls[index+1] != "syncdir" {
			t.Fatalf("%s was not followed by a directory sync: %v", call, fs.calls)
		}
	}
	if len(published) != 2 {
		t.Fatalf("published %v, want a world record then a metadata record", published)
	}
	if matched, _ := filepath.Match("slot_1_world_*.json", published[0]); !matched {
		t.Fatalf("first record published was %s, want the world", published[0])
	}
	if matched, _ := filepath.Match("slot_1_meta_*.json", published[1]); !matched {
		t.Fatalf("second record published was %s, want the metadata", published[1])
	}
}
