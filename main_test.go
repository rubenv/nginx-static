package main

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/cheekybits/is"
)

func TestTemplate(t *testing.T) {
	is := is.New(t)

	tpl, err := template.ParseFiles("nginx.conf.tpl")
	is.NoErr(err)
	is.NotNil(tpl)

	in := bytes.NewReader([]byte(`
test.example.com: /data/sites/test.example.com
`))

	cfg, err := getConfig(in)
	is.NoErr(err)
	is.NotNil(cfg)

	var out bytes.Buffer
	err = tpl.Execute(&out, cfg)
	is.NoErr(err)

	is.True(bytes.Contains(out.Bytes(), []byte("test.example.com")))
}
