package nopr

import "text/template"

const homeTmpl = `<html><head>
<title>No Pull Requests - {{.GHUser}}/{{.GHRepo}}</title>
<link rel="stylesheet" href="/static/style.css" />
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/octicons/2.2.0/octicons.css" />
</head><body><div id="container">
<h1><span class="mega-octicon octicon-git-pull-request"></span>
GitHub Pull Request Rejection Bot</h1>
<p>Some projects on GitHub don't accept GitHub Pull Requests. Maybe they have their own contribution processes. Maybe they hate freedom. Either way, GitHub doesn't provide a way to disable pull requests officially.</p>

<p>So I wrote this.</p>

<p>Using this tool, you can effectively <b>disable pull requests</b> for your repo on GitHub. When pull requests are disabled, any time a new one is opened it will immediately be closed by the bot.

<h3>Sound fun? <a href="/user">Let's get started.</a></h3>
</div><small>This project is not affiliated with GitHub.com.</small>
</body></html>`

var userTmpl = template.Must(template.New("user").Parse(`<html><head>
<title>No Pull Requests - Select a repo</title>
<link rel="stylesheet" href="/static/style.css" />
</head><body><div id="container">
<h1>Select a repo</h1><ul>
{{range .}}
  <li><a href="/repo/{{.FullName}}">{{.Owner.Login}} / {{.Name}}</a></li>
{{end}}
</ul>
<h3><a href="/">&laquo; Back</a></h3>
</div><small>This project is not affiliated with GitHub.com.</small>
</body></html>`))

// TODO: xsrf
var repoTmpl = template.Must(template.New("repo").Parse(`<html><head>
<title>No Pull Requests - {{.GHUser}}/{{.GHRepo}}</title>
<link rel="stylesheet" href="/static/style.css" />
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/octicons/2.2.0/octicons.css" />
</head><body><div id="container">
<h1><a href="https://github.com/{{.GHUser}}/{{.GHRepo}}">{{.GHUser}} / {{.GHRepo}}</a></h1>
{{if .Disabled}}
<h3>Pull requests are disabled.</h3>
<form action="/enable/{{.GHUser}}/{{.GHRepo}}" method="POST">
<button id="enable" type="submit">
  <span class="octicon octicon-git-pull-request"></span>
  Re-enable pull requests
</button>
{{else}}
<h3>Pull requests are enabled.</h3>
<form action="/disable/{{.GHUser}}/{{.GHRepo}}" method="POST">
<button id="disable" type="submit">
  <span class="octicon octicon-stop"></span>
  Disable pull requests
</button>
{{end}}
</form>
<h3><a href="/user">&laquo; Back</a></h3>
</div><small>This project is not affiliated with GitHub.com.</small>
</body></html>`))
