package nopr

import "html/template"

var userTmpl = template.Must(template.New("user").Parse(`<html><head>
<title>No Pull Requests</title>
<link rel="stylesheet" href="/static/style.css" />
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/octicons/2.2.0/octicons.css" />
</head><body><div id="container">
<h1><span class="mega-octicon octicon-git-pull-request"></span>
GitHub Pull Request Rejection Bot</h1>
<ul><form>
{{range .}}
  <li><span class="octicon octicon-repo"></span>
  <a href="https://github.com/{{.Repo.Owner.Login}}/{{.Repo.Name}}">
  {{.Repo.Owner.Login}} / <b>{{.Repo.Name}}</b></a>
{{if .Disabled}}
  <button id="enable" type="submit" formaction="/enable/{{.Repo.Owner.Login}}/{{.Repo.Name}}" formmethod="POST">
    <span class="octicon octicon-git-pull-request"></span>
    Re-enable pull requests
  </button>
{{else}}
  <button id="disable" type="submit" formaction="/disable/{{.Repo.Owner.Login}}/{{.Repo.Name}}" formmethod="POST">
    <span class="octicon octicon-stop"></span>
    Disable pull requests
  </button>
{{end}}
  </li>
{{end}}
</form></ul>
<h3><a href="/">&laquo; Home</a></h3>
</div><small>This project is not affiliated with GitHub.com.</small>


<form id="revoke" action="/revoke" method="POST">
<a href="javascript:{}" onclick="document.getElementById('revoke').submit();">Revoke access</a>
</form>
</body></html>`))

var errorTmpl = template.Must(template.New("error").Parse(`<html><head>
<title>Oh noes!!1!</title>
<link rel="stylesheet" href="/static/style.css" />
</head><body><div id="container">
<h1>Oh noes!!1!</h1>
<h3>Something went wrong.</h3>
<p>{{.}}</p>
<h3><a href="/">&laquo; Home</a></h3>
</div><small>This project is not affiliated with GitHub.com.</small>
</body></html>`))
