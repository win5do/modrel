package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func Root(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func Tags(ctx context.Context, dir string) ([]string, error) {
	out, err := run(ctx, dir, "tag", "--list")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	tags := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

func IsClean(ctx context.Context, dir string) (bool, error) {
	out, err := Status(ctx, dir)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func Status(ctx context.Context, dir string) (string, error) {
	return run(ctx, dir, "status", "--porcelain")
}

func Diff(ctx context.Context, dir string) (string, error) {
	return run(ctx, dir, "diff", "--stat")
}

func AddAll(ctx context.Context, dir string) error {
	_, err := run(ctx, dir, "add", "--all")
	return err
}

func Commit(ctx context.Context, dir string, message string) error {
	_, err := run(ctx, dir, "commit", "-m", message)
	return err
}

func Tag(ctx context.Context, dir string, tag string) error {
	_, err := run(ctx, dir, "tag", tag)
	return err
}

func PushHEAD(ctx context.Context, dir string) error {
	_, err := run(ctx, dir, "push", "origin", "HEAD")
	return err
}

func PushTag(ctx context.Context, dir string, tag string) error {
	_, err := run(ctx, dir, "push", "origin", tag)
	return err
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
