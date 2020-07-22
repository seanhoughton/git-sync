/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// git-sync is a command that pull a git repository to a local directory.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var flRepo = flag.String("repo", envString("GIT_SYNC_REPO", ""), "git repo url")
var flBranch = flag.String("branch", envString("GIT_SYNC_BRANCH", "master"), "git branch")
var flRev = flag.String("rev", envString("GIT_SYNC_REV", "HEAD"), "git rev")
var flDest = flag.String("dest", envString("GIT_SYNC_DEST", ""), "destination path")
var flWait = flag.Int("wait", envInt("GIT_SYNC_WAIT", 0), "number of seconds to wait before exit")

var metricSyncTime = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "gitsync_sync_time",
	Help:    "Sync time distributions.",
	Buckets: prometheus.ExponentialBuckets(0.01, 2, 12),
})

var metricSyncCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "gitsync_sync_count",
	Help: "Number of syncs",
})

var metricErrorCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "gitsync_error_count",
	Help: "Number of errors that have occurred",
})

func init() {
	prometheus.MustRegister(metricSyncTime)
	prometheus.MustRegister(metricSyncCount)
	prometheus.MustRegister(metricErrorCount)
}

func envString(key, def string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return def
}

func envInt(key string, def int) int {
	if env := os.Getenv(key); env != "" {
		val, err := strconv.Atoi(env)
		if err != nil {
			log.Printf("invalid value for %q: using default: %q", key, def)
			return def
		}
		return val
	}
	return def
}

const usage = "usage: GIT_SYNC_REPO= GIT_SYNC_DEST= [GIT_SYNC_BRANCH= GIT_SYNC_WAIT=] git-sync -repo GIT_REPO_URL -dest PATH [-branch -wait]"

func startMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9577", nil)
}

func main() {
	flag.Parse()
	if *flRepo == "" || *flDest == "" {
		flag.Usage()
		log.Fatal(usage)
	}
	if _, err := exec.LookPath("git"); err != nil {
		log.Fatalf("required git executable not found: %v", err)
	}

	startMetrics()

	for {
		startTime := time.Now()
		if err := syncRepo(*flRepo, *flDest, *flBranch, *flRev); err != nil {
			log.Printf("error syncing repo: %v\n", err)
			metricErrorCount.Inc()
		} else {
			metricSyncCount.Inc()
			metricSyncTime.Observe(time.Since(startTime).Seconds())
		}
		log.Printf("wait %d seconds", *flWait)
		time.Sleep(time.Duration(*flWait) * time.Second)
		log.Println("done")
	}
}

// syncRepo syncs the branch of a given repository to the destination at the given rev.
func syncRepo(repo, dest, branch, rev string) error {
	gitRepoPath := path.Join(dest, ".git")
	_, err := os.Stat(gitRepoPath)
	switch {
	case os.IsNotExist(err):
		// clone repo
		cmd := exec.Command("git", "clone", "--no-checkout", "-b", branch, repo, dest)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error cloning repo %q: %v: %s", strings.Join(cmd.Args, " "), err, string(output))
		}
		log.Printf("clone %q: %s", repo, string(output))
	case err != nil:
		return fmt.Errorf("error checking if repo exist %q: %v", gitRepoPath, err)
	}

	// fetch branch
	cmd := exec.Command("git", "fetch", "origin", branch)
	cmd.Dir = dest
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running command %q: %v: %s", strings.Join(cmd.Args, " "), err, string(output))
	}
	log.Printf("fetch %q: %s", branch, string(output))

	// reset working copy
	cmd = exec.Command("git", "reset", "--hard", rev)
	cmd.Dir = dest
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running command %q : %v: %s", strings.Join(cmd.Args, " "), err, string(output))
	}
	log.Printf("reset %q: %v", rev, string(output))

	// set file permissions
	cmd = exec.Command("chmod", "-R", "755", dest)
	cmd.Dir = dest
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running command %q : %v: %s", strings.Join(cmd.Args, " "), err, string(output))
	}

	return nil
}
