package git

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gitleaks/go-gitdiff/gitdiff"
	"github.com/rs/zerolog/log"
)

// GitLog returns a channel of gitdiff.File objects from the
// git log -p command for the given source.
func GitLog(source string, logOpts string) (<-chan *gitdiff.File, error) {
	sourceClean := filepath.Clean(source)
	var cmd *exec.Cmd
	if logOpts != "" {
		args := []string{"-C", sourceClean, "log", "-p", "-U0"}
		args = append(args, strings.Split(logOpts, " ")...)
		cmd = exec.Command("git", args...)
	} else {
		cmd = exec.Command("git", "-C", sourceClean, "log", "-p", "-U0",
			"--full-history", "--all")
	}

	log.Debug().Msgf("executing: %s", cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go listenForStdErr(stderr)
	// HACK: to avoid https://github.com/zricethezav/gitleaks/issues/722
	time.Sleep(50 * time.Millisecond)

	return gitdiff.Parse(stdout)
}

// GitDiff returns a channel of gitdiff.File objects from
// the git diff command for the given source.
func GitDiff(source string, staged bool) (<-chan *gitdiff.File, error) {
	sourceClean := filepath.Clean(source)
	var cmd *exec.Cmd
	cmd = exec.Command("git", "-C", sourceClean, "diff", "-U0", ".")
	if staged {
		cmd = exec.Command("git", "-C", sourceClean, "diff", "-U0",
			"--staged", ".")
	}
	log.Debug().Msgf("executing: %s", cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go listenForStdErr(stderr)
	// HACK: to avoid https://github.com/zricethezav/gitleaks/issues/722
	time.Sleep(50 * time.Millisecond)

	return gitdiff.Parse(stdout)
}

// listenForStdErr listens for stderr output from git and prints it to stdout
// then exits with exit code 1
func listenForStdErr(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	errEncountered := false
	for scanner.Scan() {
		// if git throws the following error:
		//
		//  exhaustive rename detection was skipped due to too many files.
		//  you may want to set your diff.renameLimit variable to at least
		//  (some large number) and retry the command.
		//
		// we skip exiting the program as git log -p/git diff will continue
		// to send data to stdout and finish executing. This next bit of
		// code prevents gitleaks from stopping mid scan if this error is
		// encountered
		if strings.Contains(scanner.Text(),
			"exhaustive rename detection was skipped") ||
			strings.Contains(scanner.Text(),
				"you may want to set your diff.renameLimit") {

			log.Warn().Msg(scanner.Text())
		} else {
			log.Error().Msg(scanner.Text())
			errEncountered = true
		}
	}
	if errEncountered {
		os.Exit(1)
	}
}
