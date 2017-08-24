package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/cheekybits/is"
)

const testCfg = `
- host: test.example.com
  root: /data/sites/test.example.com
  extra: |
    location / {
        rewrite ^/rss/(en|nl)$ /index.xml permanent;
        rewrite ^/feeds/.*\.rss$ /index.xml permanent;
    }
- host: redirect.example.com
  redirect: other.example.com
`

func TestTemplate(t *testing.T) {
	is := is.New(t)

	tpl, err := template.ParseFiles("nginx.conf.tpl")
	is.NoErr(err)
	is.NotNil(tpl)

	in := bytes.NewReader([]byte(testCfg))

	cfg, err := getConfig(in)
	is.NoErr(err)
	is.NotNil(cfg)

	var out bytes.Buffer
	err = tpl.Execute(&out, cfg)
	is.NoErr(err)

	s := out.String()
	is.True(strings.Contains(s, "server_name test.example.com"))
	is.True(strings.Contains(s, "root /data/sites/test.example.com"))
	is.True(strings.Contains(s, "/index.xml"))

	is.True(strings.Contains(s, "server_name redirect.example.com"))
	is.True(strings.Contains(s, "rewrite ^(.*)$ https://other.example.com$1 permanent"))
}
