package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"net/textproto"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jordan-wright/email"
	diffimage "github.com/schollz/go-diff-image"
	log "github.com/schollz/logger"
	"github.com/schollz/progressbar/v3"
	"github.com/yuin/goldmark"
)

var Threshold = 3
var TimeTick = 1 * time.Minute

func main() {
	err := Run()
	if err != nil {
		log.Error(err)
	}
}

var config Config

func Run() (err error) {
	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return
	}
	err = Watch(config.Watchers)
	return
}

type Config struct {
	Email    Email     `json:"email"`
	Watchers []Watcher `json:"watchers"`
}

type Email struct {
	From       string `json:"from"`
	SMTPServer string `json:"server"`
	SMTPLogin  string `json:"login"`
	SMTPPass   string `json:"pass"`
	SMTPPort   string `json:"port"`
}

type Watcher struct {
	URL         string   `json:"url"`
	CSSSelector string   `json:"css"`
	Emails      []string `json:"emails"`

	id       string
	lastFile string
}

func Watch(watchers []Watcher) (err error) {
	// download hosts file if it doesn't exist
	if !Exists("hosts") {
		log.Info("downloading hosts file")
		err = DownloadFile("http://sbc.io/hosts/alternates/fakenews-gambling-porn-social/hosts", "hosts")
		if err != nil {
			return
		}
	}
	if !Exists("package-lock.json") {
		log.Info("installing puppeteer")
		cmd := exec.Command("npm", "i", "puppeteer")
		err = cmd.Run()
		if err != nil {
			return
		}
	}

	done := make(chan bool)
	for _, watcher := range watchers {
		go func(watcher Watcher) {
			err = watcher.watch()
			if err != nil {
				done <- true
			}
		}(watcher)
	}
	<-done
	return
}

func (w *Watcher) watch() (err error) {
	if w.CSSSelector == "" {
		w.CSSSelector = "full"
	}

	h := sha1.New()
	h.Write([]byte(w.URL + w.CSSSelector))
	w.id = fmt.Sprintf("changes_%x", h.Sum(nil))
	if !Exists(w.id) {
		err = os.Mkdir(w.id, 0644)
		if err != nil {
			return
		}
	}

	// find last file
	files, err := ioutil.ReadDir(w.id)
	if err != nil {
		return
	}

	biggestNum := 1
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".png") {
			continue
		}
		num, _ := strconv.Atoi(strings.Split(f.Name(), ".")[0])
		if num > biggestNum {
			biggestNum = num
			w.lastFile = path.Join(w.id, f.Name())
			w.info(fmt.Sprintf("using last file: %s", w.lastFile))
		}
	}

	for {
		var diffFilename string
		var different bool
		diffFilename, different, err = w.capture()
		if err != nil {
			log.Errorf("error with %+v: %s", w, err.Error())
			return
		}
		if different {
			for _, to := range w.Emails {
				err = SendEmail(to, "watch "+w.URL, "site has changed on "+time.Now().String(), diffFilename)
				if err != nil {
					log.Error(err)
					return
				}
				log.Infof("%s[%s]: sent email to %s", w.URL, w.CSSSelector, to)
			}
		}
		_ = diffFilename
		time.Sleep(TimeTick)
	}
	return
}

func (w *Watcher) info(s string) {
	log.Infof("[%s|%s] %s", w.URL, w.CSSSelector, s)
}

func (w *Watcher) capture() (diffFilename string, different bool, err error) {
	w.info("capturing")

	newFile := path.Join(w.id, time.Now().Format("20060102150405.00")+".png")
	cmd := exec.Command("node", "screenshot.js", w.URL, newFile, w.CSSSelector)
	err = cmd.Run()
	if err != nil {
		log.Error(err)
		return
	}

	if Exists(w.lastFile) && Exists(newFile) {
		_, f1 := filepath.Split(w.lastFile)
		_, f2 := filepath.Split(newFile)
		diffFilename = path.Join(w.id, "diff-"+f1+"-"+f2+".jpg")
		different, err = diffImage(w.lastFile, newFile, diffFilename)
		if err != nil {
			log.Error(err)
			return
		}
		if different {
			w.info("different")
			w.lastFile = newFile
		} else {
			os.Remove(diffFilename)
			os.Remove(newFile)
			w.info("no change")
		}
	} else {
		w.lastFile = newFile
	}

	return
}

func diffImage(im1, im2, out string) (different bool, err error) {
	im1f, err := os.Open(im1)
	if err != nil {
		return
	}
	defer im1f.Close()

	img1, err := png.Decode(im1f)
	if err != nil {
		return
	}

	im2f, err := os.Open(im2)
	if err != nil {
		return
	}
	defer im2f.Close()

	img2, err := png.Decode(im2f)
	if err != nil {
		return
	}

	diffImg, _, insertions, _ := diffimage.DiffImage(img1, img2, true)
	if insertions > Threshold {
		different = true
	}
	log.Infof("found %d insertions", insertions)

	fSave, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0644)
	defer fSave.Close()
	if err != nil {
		return
	}

	err = jpeg.Encode(fSave, diffImg, &jpeg.Options{30})
	return
}

func DownloadFile(urlToGet, fileToSave string) (err error) {
	req, err := http.NewRequest("GET", urlToGet, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var out io.Writer
	f, err := os.OpenFile(fileToSave, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	out = f
	defer f.Close()
	bar := progressbar.DefaultBytes(
		int64(resp.ContentLength),
	)
	bar.RenderBlank()
	out = io.MultiWriter(out, bar)
	io.Copy(out, resp.Body)
	fmt.Print("\n")
	return
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func SendEmail(to, subject, markdown, attachment string) (err error) {
	from := config.Email.From
	smtpAuth := config.Email.SMTPLogin
	smtpPass := config.Email.SMTPPass
	SMTPHost := config.Email.SMTPServer
	SMTPPort := config.Email.SMTPPort
	if smtpAuth == "" || smtpPass == "" {
		err = fmt.Errorf("Must define environmental variables SMTPAUTH and SMTPPASS")
		return
	}

	var buf bytes.Buffer
	if err = goldmark.Convert([]byte(markdown), &buf); err != nil {
		return
	}

	e := &email.Email{
		To:      []string{to},
		From:    from,
		Subject: subject,
		Text:    []byte(markdown),
		HTML:    buf.Bytes(),
		Headers: textproto.MIMEHeader{},
	}
	if attachment != "" {
		e.AttachFile(attachment)
	}
	err = e.SendWithTLS(SMTPHost+":"+SMTPPort, smtp.PlainAuth("", smtpAuth, smtpPass, SMTPHost), &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         SMTPHost,
	})
	return
}
