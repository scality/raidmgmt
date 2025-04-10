package megaraid

// PathResolver is an interface that defines the EvalSymlinks method.
//
// FileExists checks if a file exists at the given path.
// It is used to mock the os package in tests.
// EvalSymlinks returns the path name after the evaluation of any symbolic links.
// It is used to mock the filepath.EvalSymlinks function in tests.
type PathResolver interface {
	FileExists(string) bool
	EvalSymlinks(string) (string, error)
}
