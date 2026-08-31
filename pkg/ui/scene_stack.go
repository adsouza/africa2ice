package ui

type SceneID uint8

const (
	SceneGameplay SceneID = iota
	ScenePause
	SceneStorage
	SceneSettings
)

// SceneStack is bounded presentation state. It never contains a frame or
// simulation value; overlays can be pushed and dismissed without mutating the
// accepted world.
type SceneStack struct {
	items [4]SceneID
	depth int
}

func NewSceneStack() SceneStack {
	return SceneStack{items: [4]SceneID{SceneGameplay}, depth: 1}
}

func (stack *SceneStack) Current() SceneID {
	if stack == nil || stack.depth == 0 {
		return SceneGameplay
	}
	return stack.items[stack.depth-1]
}

func (stack *SceneStack) Push(scene SceneID) bool {
	if stack == nil || stack.depth == len(stack.items) || stack.Current() == scene {
		return false
	}
	stack.items[stack.depth] = scene
	stack.depth++
	return true
}

func (stack *SceneStack) Pop() bool {
	if stack == nil || stack.depth <= 1 {
		return false
	}
	stack.depth--
	stack.items[stack.depth] = 0
	return true
}

func (stack *SceneStack) Reset() {
	if stack == nil {
		return
	}
	*stack = NewSceneStack()
}
