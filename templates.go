package nopr

import "html/template"

const homeTmpl = `<html><body><h1>Pull Request Rejection Bot</h1>
<p>Some projects on GitHub don't accept GitHub Pull Requests. Maybe they have their own contribution processes. Maybe they hate freedom. Either way, GitHub doesn't provide a way to disable pull requests officially. So I wrote this.</p>

<p>Using this tool you can effectively disable pull requests for your repo on GitHub. When pull requests are disabled, any time a new one is opened it will immediately be closed by the bot.

<p>Sound fun? <a href="/user">Let's get started.</a>
</body></html>`

var userTmpl = template.Must(template.New("user").Parse(`<html><body><h1>Select a repo</h1><ul>
{{range .}}
  <li><a href="/repo/{{.FullName}}">{{.FullName}}</a></li>
{{end}}
</ul></body></html>`))

// TODO: xsrf
var repoTmpl = template.Must(template.New("repo").Parse(`<html><body>
{{if .Disabled}}
<h1>Pull requests are disabled</h1>
<form action="/enable/{{.FullName}}" method="POST">
Click to re-enable pull requests for {{.FullName}}
<input type="submit" value="Re-enable pull requests"></input>
{{else}}
<h1>Pull requests are enabled</h1>
<form action="/disable/{{.FullName}}" method="POST">
Click to disable pull requests for {{.FullName}}
<input type="submit" value="Disable pull requests"></input>
{{end}}
</form></body></html>`))
