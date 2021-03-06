/*
Copyright 2019 The Kubernetes Authors.

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

package updater

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
)

type updateClient interface {
	UpdatePullRequest(org, repo string, number int, title, body *string, open *bool, branch *string, canModify *bool) error
	BotName() (string, error)
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
}

type ensureClient interface {
	updateClient
	CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (int, error)
}

func UpdatePR(org, repo, title, body, matchTitle string, gc updateClient) (*int, error) {
	if matchTitle == "" {
		return nil, nil
	}

	logrus.Info("Looking for a PR to reuse...")
	me, err := gc.BotName()
	if err != nil {
		return nil, fmt.Errorf("bot name: %v", err)
	}

	issues, err := gc.FindIssues("is:open is:pr archived:false in:title author:"+me+" "+matchTitle, "updated", false)
	if err != nil {
		return nil, fmt.Errorf("find issues: %v", err)
	} else if len(issues) == 0 {
		logrus.Info("No reusable issues found")
		return nil, nil
	}
	n := issues[0].Number
	logrus.Infof("Found %d", n)
	var ignoreOpen *bool
	var ignoreBranch *string
	var ignoreModify *bool
	if err := gc.UpdatePullRequest(org, repo, n, &title, &body, ignoreOpen, ignoreBranch, ignoreModify); err != nil {
		return nil, fmt.Errorf("update %d: %v", n, err)
	}

	return &n, nil
}

func EnsurePR(org, repo, title, body, source, branch, matchTitle string, gc ensureClient) (*int, error) {
	n, err := UpdatePR(org, repo, title, body, matchTitle, gc)
	if err != nil {
		return nil, fmt.Errorf("update error: %v", err)
	}
	if n == nil {
		allowMods := true
		pr, err := gc.CreatePullRequest(org, repo, title, body, source, branch, allowMods)
		if err != nil {
			return nil, fmt.Errorf("create error: %v", err)
		}
		n = &pr
	}
	return n, nil
}
