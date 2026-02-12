package ui

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func DeleteBranches(ctx context.Context, branches []string) []string {
	var deleted = make([]string, 0, len(branches))
	for _, branch := range branches {
		cmd := exec.CommandContext(ctx, "git", "branch", "-D", branch) //nolint:gosec // want to run git command
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Run()
		deleted = append(deleted, branch)
	}
	return deleted
}

func GetBranches(path string) ([]BranchInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Local branches
	cmd := exec.CommandContext(ctx, "git", "branch", "--format=%(refname:short)")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Use bytes.Fields on []byte to avoid extra string allocation
	bFields := bytes.Fields(out)
	localBranches := make([]string, 0, len(bFields))
	for _, f := range bFields {
		localBranches = append(localBranches, string(f))
	}

	// Remote branches
	cmd = exec.CommandContext(ctx, "git", "branch", "-r", "--format=%(refname:short)")
	cmd.Dir = path
	out, err = cmd.Output()
	if err != nil {
		return nil, err
	}

	remoteBranches := make(map[string]struct{})
	rFields := bytes.Fields(out) //nolint:modernize // small, explicit allocation is acceptable here
	for _, f := range rFields {
		b := string(f)
		parts := strings.SplitN(b, "/", 2)
		if len(parts) == 2 {
			remoteBranches[parts[1]] = struct{}{}
		}
	}

	var result = make([]BranchInfo, 0, len(localBranches))

	for _, branch := range localBranches {
		notOnRemote := true
		if _, ok := remoteBranches[branch]; ok {
			notOnRemote = false
		}

		// Check if branch has commits not on any remote
		hasUniqueCommits := false
		cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", branch, "--not", "--remotes=origin") //nolint:gosec // want to run git command
		cmd.Dir = path
		out, err = cmd.Output()
		if err == nil {
			count := strings.TrimSpace(string(out))
			if count != "0" {
				hasUniqueCommits = true
			}
		}

		// Check if branch is behind main
		isBehindMain := false
		cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", "main.."+branch) //nolint:gosec // want to run git command
		cmd.Dir = path
		out, err = cmd.Output()
		if err == nil {
			count := strings.TrimSpace(string(out))
			if count == "0" {
				// branch is not ahead of main, so check if main is ahead
				cmd2 := exec.CommandContext(ctx, "git", "rev-list", "--count", branch+"..main") //nolint:gosec // want to run git command
				cmd2.Dir = path
				out2, err2 := cmd2.Output()
				if err2 == nil {
					count2 := strings.TrimSpace(string(out2))
					if count2 != "0" {
						isBehindMain = true
					}
				}
			}
		}

		// Last commit time
		cmd = exec.CommandContext(ctx, "git", "log", "-1", "--format=%ct", branch) //nolint:gosec // want to run git command
		cmd.Dir = path
		out, _ = cmd.Output()
		var t time.Time
		if len(out) > 0 {
			sec, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
			t = time.Unix(sec, 0)
		}

		result = append(result, BranchInfo{
			Name:             branch,
			NotOnRemote:      notOnRemote,
			HasUniqueCommits: hasUniqueCommits,
			IsBehindMain:     isBehindMain,
			LastCommit:       t,
		})
	}

	return result, nil
}
