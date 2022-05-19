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
	_, err := getReleaseBranch()
	if err == nil {
		fmt.Println("リリースブランチが存在しません！")
		fmt.Println(err)
		return
	}

	pullRequests, err := pullRequestList(nil)

	if err != nil {
		fmt.Println("失敗 #1. リリースするプルリクエストを取得")
		fmt.Println(err)
		return
	}

	releasePullRquests := releasePullRquestList(pullRequests)

	if len(releasePullRquests) == 0 {
		fmt.Println("対象のプルリクエストが存在しません")
		return
	}
	fmt.Println("完了 #1. リリースするプルリクエストを取得")

	err = mergeBlanch(releasePullRquests)

	if err != nil {
		fmt.Println("失敗 #2. リリースブランチにマージ")
		fmt.Println(err)
		return
	}
	fmt.Println("完了 #2. リリースブランチにマージ")

	if isReleasePullRequestExsit(releasePullRquests) {
		_, err := createRleasePullRequest(releasePullRquests)

		if err != nil {
			fmt.Println("失敗 #3. プルリクエスト作成")
			fmt.Println(err)
			return
		}

		fmt.Println("完了 #3. プルリクエスト作成")
	} else {
		_, err := updateRleasePullRequest(releasePullRquests)

		if err != nil {
			fmt.Println("失敗 #3. プルリクエスト更新")
			fmt.Println(err)
			return
		}

		fmt.Println("完了 #3. プルリクエスト更新")
	}
}

func githubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func getReleaseBranch() (*github.Branch, error) {
	client := githubClient()
	ctx := context.Background()

	b, _, err := client.Repositories.GetBranch(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), os.Args[0])
	return b, err

}

func pullRequestList(opt *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	pulls, _, err := client.PullRequests.List(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), opt)

	return pulls, err

}

func releasePullRquestList(pulls []*github.PullRequest) []*github.PullRequest {
	var releasePulls []*github.PullRequest

	for _, pr := range pulls {
		if strings.Contains(*pr.Title, "【定期リリース】") {
			fmt.Println(*pr.Title)
			continue
		}

		for _, label := range pr.Labels {
			if *label.Name == os.Args[0] {
				releasePulls = append(releasePulls, pr)
			}
		}
	}

	return releasePulls
}

func mergeBlanch(pulls []*github.PullRequest) error {
	client := githubClient()
	ctx := context.Background()
	pullsSize := len(pulls)

	for i, pr := range pulls {
		fmt.Printf("マージ開始[%s: %b/%b]\n", *pr.Head.Ref, i+1, pullsSize)

		req := &github.RepositoryMergeRequest{
			Base: pr.Base.Ref,
			Head: pr.Head.Ref,
		}
		_, _, err := client.Repositories.Merge(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), req)

		if err != nil {
			return err
		}

		fmt.Printf("マージ完了[%s: %b/%b]\n", *pr.Head.Ref, i+1, pullsSize)
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

func createRleasePullRequest(pulls []*github.PullRequest) (*github.PullRequest, error) {
	client := githubClient()
	ctx := context.Background()

	title := releasePullRquestTitle()
	body := releasePullRquestBody(pulls)
	base := os.Getenv("BASEBRANCH")
	head := os.Args[0]

	req := &github.NewPullRequest{
		Title: &title,
		Base:  &base,
		Head:  &head,
		Body:  &body,
	}

	pr, _, err := client.PullRequests.Create(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), req)
	return pr, err
}

func updateRleasePullRequest(pulls []*github.PullRequest) (*github.PullRequest, error) {
	client := githubClient()
	ctx := context.Background()

	title := releasePullRquestTitle()
	body := releasePullRquestBody(pulls)

	for _, pr := range pulls {
		if strings.Contains(*pr.Title, "【定期リリース】") {
			pr.Title = &title
			pr.Body = &body

			pr, _, err := client.PullRequests.Edit(ctx, os.Getenv("OWNER"), os.Getenv("REPO"), *pr.Number, pr)
			return pr, err
		}
	}

	return nil, nil
}

func releasePullRquestTitle() string {
	title := "【定期リリース】" + os.Args[0]
	return title
}

func releasePullRquestBody(pulls []*github.PullRequest) string {
	body := "## リリースプルリク一覧\n"
	for _, pr := range pulls {
		body += fmt.Sprintf("%v\n%v\n%v\n\n", *pr.Title, *pr.HTMLURL, *pr.Head.Ref)
	}

	return body
}
