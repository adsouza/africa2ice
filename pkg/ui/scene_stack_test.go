package ui

import "testing"

func TestSceneStackIsBoundedAndKeepsGameplayAsItsRoot(t *testing.T) {
	stack := NewSceneStack()
	if stack.Current() != SceneGameplay || stack.Pop() {
		t.Fatal("new stack did not retain gameplay root")
	}
	if !stack.Push(SceneMenu) || !stack.Push(SceneStorage) || stack.Current() != SceneStorage {
		t.Fatalf("pushed stack = %#v", stack)
	}
	if !stack.Push(SceneSettings) || stack.Current() != SceneSettings {
		t.Fatalf("third overlay push = %#v", stack)
	}
	if stack.Push(SceneMenu) {
		t.Fatal("scene stack exceeded its fixed four-entry capacity")
	}
	stack.Pop()
	if stack.Push(SceneStorage) {
		t.Fatal("duplicate top scene was pushed")
	}
	if !stack.Pop() || stack.Current() != SceneMenu {
		t.Fatalf("pop = %#v", stack)
	}
	stack.Reset()
	if stack.Current() != SceneGameplay || stack.depth != 1 {
		t.Fatalf("reset stack = %#v", stack)
	}
}
