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
</body>
<a href="https://github.com/imjasonh/nopullrequests"><img style="position: absolute; top: 0; right: 0; border: 0;" src="https://camo.githubusercontent.com/38ef81f8aca64bb9a64448d0d70f1308ef5341ab/68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f6769746875622f726962626f6e732f666f726b6d655f72696768745f6461726b626c75655f3132313632312e706e67" alt="Fork me on GitHub" data-canonical-src="https://s3.amazonaws.com/github/ribbons/forkme_right_darkblue_121621.png"></a>
</html>`))

var errorTmpl = template.Must(template.New("error").Parse(`<html><head>
<title>Oh noes!!1!</title>
<link rel="stylesheet" href="/static/style.css" />
</head><body><div id="container">
<h1>Oh noes!!1!</h1>
<h3>Something went wrong.</h3>
<p>{{.}}</p>
<h3><a href="/">&laquo; Home</a></h3>
</div><small>This project is not affiliated with GitHub.com.</small>
</body>
<a href="https://github.com/imjasonh/nopullrequests"><img style="position: absolute; top: 0; right: 0; border: 0;" src="https://camo.githubusercontent.com/38ef81f8aca64bb9a64448d0d70f1308ef5341ab/68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f6769746875622f726962626f6e732f666f726b6d655f72696768745f6461726b626c75655f3132313632312e706e67" alt="Fork me on GitHub" data-canonical-src="https://s3.amazonaws.com/github/ribbons/forkme_right_darkblue_121621.png"></a>
</html>`))
