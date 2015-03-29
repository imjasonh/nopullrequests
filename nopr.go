// TODO: allow users to configure behavior:
// - whether to close the PR or add a status (closing hides statuses)
// - whether to comment on the PR before closing
// - custom text to use when closing
// TODO: add link to revoke token and remove hooks
// TODO: use appengine-value to store client secret
// TODO: use gorilla sessions instead of Google auth
// TODO: xsrf everywhere

package nopr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"

	"github.com/google/go-github/github"
)

const (
	// TODO: store these more securely (and revoke these when you do!)
	clientID        = "350be49c3c1988aac719"
	clientSecret    = "f14c9383c4b8964781ea4acdd881946b1dfed488"
	redirectURLPath = "/oauthcallback"
)

var scopes = strings.Join([]string{
	"user:email",      // permission to get basic information about the user
	"public_repo",     // permission to close PRs
	"admin:repo_hook", // permission to add/delete webhooks
	// TODO: ask for this when we're not just closing the PR
	// "repo:status",     // permission to add statuses to commits
}, ",")

func init() {
	http.HandleFunc("/start", startHandler)
	http.HandleFunc(redirectURLPath, oauthHandler)
	http.HandleFunc("/user", userHandler)
	http.HandleFunc("/enable/", enableHandler)
	http.HandleFunc("/disable/", disableHandler)
	http.HandleFunc("/hook", webhookHandler)
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		ctx.Infof("not logged in, redirecting...")
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}

	ctx.Infof("starting oauth...")
	redirectURL := fmt.Sprintf("https://%s.appspot.com", appengine.AppID(ctx)) + redirectURLPath
	url := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s",
		clientID, redirectURL, scopes)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func renderError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	errorTmpl.Execute(w, msg)
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	code := r.FormValue("code")
	if code == "" {
		ctx.Errorf("no code, going to start")
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}

	u := user.Current(ctx)
	if u == nil {
		ctx.Infof("not logged in, redirecting...")
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}

	tok, err := getAccessToken(ctx, code)
	if err != nil {
		ctx.Errorf("getting access token: %v", err)
		renderError(w, "Error getting access token")
		return
	}

	ghu, _, err := newClient(ctx, tok).Users.Get("")
	if err != nil {
		ctx.Errorf("getting user: %v", err)
		renderError(w, "Error getting user")
		return
	}

	if err := PutUser(ctx, User{
		GoogleUserID: u.ID,
		GitHubUserID: *ghu.ID,
		GitHubToken:  tok,
	}); err != nil {
		ctx.Errorf("put user: %v", err)
		renderError(w, "Error writing user entry")
		return
	}
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func getAccessToken(ctx appengine.Context, code string) (string, error) {
	client := urlfetch.Client(ctx)
	url := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s",
		clientID, clientSecret, code)
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		ctx.Errorf("exchanging code: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	var b struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		ctx.Errorf("decoding json: %v", err)
		return "", err
	}
	return b.AccessToken, nil
}

func newClient(ctx appengine.Context, tok string) *github.Client {
	return github.NewClient(&http.Client{Transport: transport{ctx, tok}})
}

type transport struct {
	ctx appengine.Context
	tok string
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "token "+t.tok)
	return urlfetch.Client(t.ctx).Do(req)
}

type User struct {
	GoogleUserID string
	GitHubUserID int
	GitHubToken  string
}

func PutUser(ctx appengine.Context, u User) error {
	k := datastore.NewKey(ctx, "User", u.GoogleUserID, 0, nil)
	_, err := datastore.Put(ctx, k, &u)
	return err
}

func GetUser(ctx appengine.Context, id string) *User {
	k := datastore.NewKey(ctx, "User", id, 0, nil)
	var u User
	if err := datastore.Get(ctx, k, &u); err == datastore.ErrNoSuchEntity {
		return nil
	} else if err != nil {
		ctx.Errorf("getting user: %v", err)
		return nil
	}
	return &u
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	uu := user.Current(ctx)
	if uu == nil {
		ctx.Infof("not logged in, redirecting...")
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	u := GetUser(ctx, uu.ID)
	if u == nil {
		ctx.Infof("unknown user, going to /start")
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}

	repos, _, err := newClient(ctx, u.GitHubToken).Repositories.List("", &github.RepositoryListOptions{
		Type: "admin",
	})
	if err != nil {
		ctx.Errorf("listing repos: %v", err)
		renderError(w, "Error listing repos")
		return
	}

	type data struct {
		Repo     github.Repository
		Disabled bool
	}
	d := []data{}

	keys := []*datastore.Key{}
	for _, r := range repos {
		keys = append(keys, datastore.NewKey(ctx, "Repo", *r.FullName, 0, nil))
	}
	repoEntities := make([]Repo, len(keys))
	if err := datastore.GetMulti(ctx, keys, repoEntities); err != nil {
		if me, ok := err.(appengine.MultiError); ok {
			for i, e := range me {
				var disabled = e == nil
				d = append(d, data{Repo: repos[i], Disabled: disabled})
			}
		} else {
			ctx.Errorf("getmulti: %v", err)
			renderError(w, "Error retrieving repos")
			return
		}
	} else {
		// all repos are disabled
		for _, r := range repos {
			d = append(d, data{Repo: r, Disabled: true})
		}
	}

	if err := userTmpl.Execute(w, d); err != nil {
		ctx.Errorf("executing template: %v", err)
	}
}

type Repo struct {
	FullName  string // e.g., MyUser/foo-bar
	UserID    string // User key to use to close PRs
	WebhookID int    // Used to delete the hook
}

func (r Repo) Split() (string, string) {
	parts := strings.Split(r.FullName, "/")
	if len(parts) < 2 {
		panic("invalid full name: " + r.FullName)
	}
	return parts[0], parts[1]
}

func PutRepo(ctx appengine.Context, r Repo) error {
	k := datastore.NewKey(ctx, "Repo", r.FullName, 0, nil)
	_, err := datastore.Put(ctx, k, &r)
	return err
}

func GetRepo(ctx appengine.Context, fn string) *Repo {
	k := datastore.NewKey(ctx, "Repo", fn, 0, nil)
	var r Repo
	if err := datastore.Get(ctx, k, &r); err == datastore.ErrNoSuchEntity {
		return nil
	} else if err != nil {
		ctx.Errorf("getting repo: %v", err)
		return nil
	}
	return &r
}

func DeleteRepo(ctx appengine.Context, fn string) error {
	return datastore.Delete(ctx, datastore.NewKey(ctx, "Repo", fn, 0, nil))
}

func disableHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}

	ctx := appengine.NewContext(r)
	uu := user.Current(ctx)
	if uu == nil {
		ctx.Infof("not logged in, redirecting...")
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	u := GetUser(ctx, uu.ID)
	if u == nil {
		ctx.Infof("unknown user, going to /start")
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}
	// TODO: check that the user is an admin on the repo

	fullName := r.URL.Path[len("/disable/"):]

	ghUser, ghRepo := Repo{FullName: fullName}.Split()
	hook, _, err := newClient(ctx, u.GitHubToken).Repositories.CreateHook(ghUser, ghRepo, &github.Hook{
		Name:   github.String("web"),
		Events: []string{"pull_request"},
		Config: map[string]interface{}{
			"content_type": "json",
			"url":          fmt.Sprintf("https://%s.appspot.com/hook", appengine.AppID(ctx)),
		},
	})
	if err != nil {
		ctx.Errorf("creating hook: %v", err)
		renderError(w, "Error creating webhook")
		return
	}

	if err := PutRepo(ctx, Repo{
		FullName:  fullName,
		UserID:    u.GoogleUserID,
		WebhookID: *hook.ID,
	}); err != nil {
		ctx.Errorf("put repo: %v", err)
		renderError(w, "Error writing repo entry")
		return
	}
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func enableHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}

	ctx := appengine.NewContext(r)
	uu := user.Current(ctx)
	if uu == nil {
		ctx.Infof("not logged in, redirecting...")
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	u := GetUser(ctx, uu.ID)
	if u == nil {
		ctx.Infof("unknown user, going to /start")
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}
	// TODO: check that the user is an admin on the repo

	fullName := r.URL.Path[len("/enable/"):]

	repo := GetRepo(ctx, fullName)
	if repo == nil {
		http.Error(w, "repo not found", http.StatusNotFound)
		return
	}

	ghUser, ghRepo := repo.Split()
	if _, err := newClient(ctx, u.GitHubToken).Repositories.DeleteHook(ghUser, ghRepo, repo.WebhookID); err != nil {
		ctx.Errorf("delete hook: %v", err)
		renderError(w, "Error deleting webhook")
		return
	}
	if err := DeleteRepo(ctx, repo.FullName); err != nil {
		ctx.Errorf("delete repo: %v", err)
		renderError(w, "Error deleting repo entry")
		return
	}
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if r.Method != "POST" {
		return
	}
	if r.Header.Get("X-Github-Event") != "pull_request" {
		return
	}
	defer r.Body.Close()
	var hook github.PullRequestEvent
	if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
		ctx.Errorf("decoding json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if *hook.Action != "opened" && *hook.Action != "reopened" {
		return
	}
	ctx.Infof("got webhook for pull request %d opened for %q (%s)", *hook.Number, *hook.Repo.FullName, *hook.PullRequest.Head.SHA)

	repo := GetRepo(ctx, *hook.Repo.FullName)
	if repo == nil {
		ctx.Errorf("unknown repo")
		// TODO: delete webhook?
		return
	}

	user := GetUser(ctx, repo.UserID)
	if user == nil {
		ctx.Errorf("unknown user %q", repo.UserID)
		// TODO: user who configured the hook has left?
		return
	}

	ghUser, ghRepo := repo.Split()
	client := newClient(ctx, user.GitHubToken)

	// TODO: Commit statuses are hidden when the PR is closed, and stick around
	// once they're reopened. Either the PR should stay open with a failed status,
	// and the status should be removed when PRs are re-enabled (ugh), or we can
	// just skip the status and comment and close.
	/*
		if _, _, err := client.Repositories.CreateStatus(ghUser, ghRepo, *hook.PullRequest.Head.SHA, &github.RepoStatus{
			State:       github.String("error"),
			TargetURL:   github.String("https://nopullrequests.appspot.com"),
			Description: github.String("This repository has chosen not to enable pull requests."), // TODO: configurable
			Context:     github.String("no pull requests"),
		}); err != nil {
			ctx.Errorf("failed to create status on %q: %v", *hook.PullRequest.Head.SHA, err)
		}
	*/

	if _, _, err := client.Issues.CreateComment(ghUser, ghRepo, *hook.Number, &github.IssueComment{
		Body: github.String("This repository has chosen to disable pull requests."), // TODO: configurable
	}); err != nil {
		ctx.Errorf("failed to create comment: %v", err)
	}

	// TODO: this seems to hide the commit status, maybe this should post a comment instead?
	if _, _, err := client.PullRequests.Edit(ghUser, ghRepo, *hook.Number, &github.PullRequest{
		State: github.String("closed"),
	}); err != nil {
		ctx.Errorf("failed to close pull request: %v", err)
	}
}
