package commandrunner

type CommandRunner interface {
	Run(args []string) ([]byte, error)
}
