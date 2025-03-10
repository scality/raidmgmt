package megaraid

// PathResolver is an interface that defines the EvalSymlinks method.
//
// EvalSymlinks returns the path name after the evaluation of any symbolic links.
// It is used to mock the filepath.EvalSymlinks function in tests.
type PathResolver interface {
	EvalSymlinks(string) (string, error)
}
