package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	if _, err := getReleaseBranch(); err != nil {
		if _, err := createNewBranch(); err != nil {
			fmt.Println("リリースブランチ作成失敗!!")
			fmt.Println(err)
			return
		}
	}

	pullRequests, err := pullRequestList(nil)
	if err != nil {
		fmt.Println("失敗 #1. リリースするプルリクエストを取得")
		fmt.Println(err)
		return
	}

	releasePullRequests := releasePullRequestList(pullRequests)

	if len(releasePullRequests) == 0 {
		fmt.Println("対象のプルリクエストが存在しません")
		return
	}
	fmt.Println("完了 #1. リリースするプルリクエストを取得")

	err = mergeBlanch(releasePullRequests)
	if err != nil {
		fmt.Println("失敗 #2. リリースブランチにマージ")
		fmt.Println(err)
		return
	}
	fmt.Println("完了 #2. リリースブランチにマージ")

	if isReleasePullRequestExsit(pullRequests) {
		_, err := updateReleasePullRequest(releasePullRequests)

		if err != nil {
			fmt.Println("失敗 #3. プルリクエスト更新")
			fmt.Println(err)
			return
		}

		fmt.Println("完了 #3. プルリクエスト更新")
	} else {
		_, err := createReleasePullRequest(releasePullRequests)

		if err != nil {
			fmt.Println("失敗 #3. プルリクエスト作成")
			fmt.Println(err)
			return
		}

		fmt.Println("完了 #3. プルリクエスト作成")
	}
}

func githubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	c := github.NewClient(tc)

	return c
}

func getReleaseBranch() (*github.Branch, error) {
	c := githubClient()
	ctx := context.Background()

	b, _, err := c.Repositories.GetBranch(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), os.Args[1])
	return b, err
}

func getLatestMainref() (*github.Reference, error) {
	c := githubClient()
	ctx := context.Background()

	ref, _, err := c.Git.GetRef(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), "heads/main")

	return ref, err
}

func createNewBranch() (*github.Reference, error) {
	c := githubClient()
	ctx := context.Background()
	mainRef, err := getLatestMainref()

	if err != nil {
		return nil, err
	}
	newRef := "refs/heads/" + os.Args[1]
	obj := &github.GitObject{SHA: mainRef.Object.SHA}
	ref := &github.Reference{Ref: &newRef, Object: obj}

	r, _, err := c.Git.CreateRef(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), ref)
	return r, err
}

func pullRequestList(opt *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	c := github.NewClient(tc)
	pulls, _, err := c.PullRequests.List(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), opt)

	return pulls, err

}

func releasePullRequestList(pulls []*github.PullRequest) []*github.PullRequest {
	var releasePulls []*github.PullRequest

	for _, pr := range pulls {
		if strings.Contains(*pr.Title, "【定期リリース】") {
			continue
		}

		for _, label := range pr.Labels {
			if *label.Name == os.Args[1] {
				releasePulls = append(releasePulls, pr)
			}
		}
	}

	return releasePulls
}

func mergeBlanch(pulls []*github.PullRequest) error {
	c := githubClient()
	ctx := context.Background()

	for _, pr := range pulls {
		fmt.Printf("マージ開始[%s]\n", *pr.Head.Ref)
		req := &github.RepositoryMergeRequest{
			Base: &os.Args[1],
			Head: pr.Head.Ref,
		}
		_, _, err := c.Repositories.Merge(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), req)

		if err != nil {
			return err
		}
	}

	return nil
}

func isReleasePullRequestExsit(pulls []*github.PullRequest) bool {
	for _, pr := range pulls {
		if strings.Contains(*pr.Title, "【定期リリース】") {
			return true
		}
	}

	return false
}

func createReleasePullRequest(pulls []*github.PullRequest) (*github.PullRequest, error) {
	c := githubClient()
	ctx := context.Background()

	title := releasePullRequestTitle()
	body := releasePullRequestBody(pulls)
	base := os.Getenv("BASEBRANCH")
	head := os.Args[1]

	req := &github.NewPullRequest{
		Title: &title,
		Base:  &base,
		Head:  &head,
		Body:  &body,
	}

	pr, _, err := c.PullRequests.Create(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), req)
	return pr, err
}

func updateReleasePullRequest(pulls []*github.PullRequest) (*github.PullRequest, error) {
	c := githubClient()
	ctx := context.Background()

	title := releasePullRequestTitle()
	body := releasePullRequestBody(pulls)

	for _, pr := range pulls {
		if strings.Contains(*pr.Title, "【定期リリース】") {
			pr.Title = &title
			pr.Body = &body

			pr, _, err := c.PullRequests.Edit(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), *pr.Number, pr)
			return pr, err
		}
	}

	return nil, nil
}

func releasePullRequestTitle() string {
	return "【定期リリース】" + os.Args[1]
}

func releasePullRequestBody(pulls []*github.PullRequest) string {
	body := "## リリースプルリク一覧\n"
	for _, pr := range pulls {
		body += fmt.Sprintf("%v\n%v\n%v\n\n", *pr.Title, *pr.HTMLURL, *pr.Head.Ref)
	}

	return body
}
