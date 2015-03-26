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
	clientID        = "TODO"
	clientSecret    = "TODO"
	redirectURLPath = "/oauthcallback"
)

var scopes = strings.Join([]string{
	"user:email",      // permission to get basic information about the user
	"repo:status",     // permission to add statuses to commits
	"write:repo_hook", // permission to add a webhook to repos
}, ",")

func init() {
	http.HandleFunc("/start", startHandler)
	http.HandleFunc(redirectURLPath, oauthHandler)
	http.HandleFunc("/user", userHandler)
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}

	host := fmt.Sprintf("https://%s.appspot.com", appengine.AppID(ctx))
	if appengine.IsDevAppServer() {
		host = "http://localhost:8080"
	}
	redirectURL := host + redirectURLPath
	url := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s",
		clientID, redirectURL, scopes)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	if code == "" {
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}

	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}

	tok, err := getAccessToken(ctx, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ghu, _, err := newClient(ctx, tok).Users.Get("")
	if err != nil {
		ctx.Errorf("getting user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := PutUser(ctx, User{
		GoogleUserID: u.ID,
		GitHubUserID: *ghu.ID,
		GitHubToken:  tok,
	}); err != nil {
		ctx.Errorf("put user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func GetUser(ctx appengine.Context, uu *user.User) *User {
	k := datastore.NewKey(ctx, "User", uu.ID, 0, nil)
	var u *User
	if err := datastore.Get(ctx, k, u); err == datastore.ErrNoSuchEntity {
		return nil
	} else if err != nil {
		ctx.Errorf("getting user: %v", err)
		return nil
	}
	return u
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	uu := user.Current(ctx)
	if uu == nil {
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	u := GetUser(ctx, uu)
	if u == nil {
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}

	repos, _, err := newClient(ctx, u.GitHubToken).Repositories.List("", &github.RepositoryListOptions{
		Type: "admin",
	})
	if err != nil {
		ctx.Errorf("listing repos: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "text/html")
	fmt.Fprintln(w, "<html><body><ul>")
	for _, r := range repos {
		fmt.Fprintf(w, `<li><a href="/repo/%s">%s</a></li>\n`, *r.FullName, *r.FullName)
	}
	fmt.Fprintln(w, "</ul></body></html>")
}

func repoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	uu := user.Current(ctx)
	if uu == nil {
		loginURL, _ := user.LoginURL(ctx, r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	u := GetUser(ctx, uu)
	if u == nil {
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}

	fullName := r.URL.Path[len("/repo/"):]
	fmt.Fprintln(w, fullName)

	parts := strings.Split(fullName, "/")
	repoUser := parts[0]
	repoName := parts[1]

	newClient(ctx, u.GitHubToken).Repositories.Get(repoUser, repoName)
	// TODO: get the repo, check the user is an admin
	// TODO: display current repo config, allow updates to config (enable/disable, specific message)
}

type Repo struct {
	FullName string // e.g., MyUser/foo-bar
}

func (r Repo) Split() (string, string) {
	parts := strings.Split(r.FullName, "/")
	if len(parts) != 2 {
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
	var r *Repo
	if err := datastore.Get(ctx, k, r); err == datastore.ErrNoSuchEntity {
		return nil
	} else if err != nil {
		ctx.Errorf("getting repo: %v", err)
		return nil
	}
	return r
}
