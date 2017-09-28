package main

import (
	"encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/ChimeraCoder/anaconda"
)

// Obj ...
// event, t.CreatedAt, t.User.ScreenName, source, t.User.Location, text
type Obj struct {
	Event      string `csv:"event"`
	CreatedAt  string `csv:"created_at"`
	ScreenName string `csv:"screen_name"`
	Source     string `csv:"source"`
	Location   string `csv:"location"`
	Text       string `csv:"text"`
}

// Credentials ..
type Credentials struct {
	ConsumerKey       string `yaml:"consumer-key"`
	ConsumerSecret    string `yaml:"consumer-secret"`
	AccessToken       string `yaml:"access-token"`
	AccessTokenSecret string `yaml:"access-token-secret"`
}

var (
	// Creds ...
	Creds struct {
		Stream *Credentials `yaml:"stream"`
		List   *Credentials `yaml:"list"`
	}
	// Path : Path to Output Folder
	Path string
	// User : User Screen Name to Monitor Activity
	User string
	// Command : Command to Run
	Command string
	// Conf : Path to Configuration File
	Conf string
)

func init() {
	// Assign CLI Flags
	flag.StringVar(&Path, "path", "./data", "Path to Output Folder")
	flag.StringVar(&Conf, "conf", "./conf/bot.yaml", "Path to Configuration File")
	flag.StringVar(&User, "user", "", "User Screen Name to Monitor Activity")

	// Pasre CLI Flags
	flag.Parse()

	// Get Command Argument
	args := flag.Args()
	if len(args) == 0 {
		log.Println("You need to pass a command. Available Commands [stream, list, generate, post].")
		os.Exit(1)
	}

	Command = args[0]

	data, err := ioutil.ReadFile(Conf)
	if err != nil {
		log.Fatalf("parseconf error: %v", err)
	}

	err = yaml.Unmarshal(data, &Creds)
	if err != nil {
		log.Fatalf("parseconf error: %v", err)
	}
}

func main() {
	switch Command {
	case "stream":
		Stream()
	case "post":
		Post()
	case "list":
		List()
	case "generate":
		Generate()
	}
}

// Stream : Stream User's Related Activity
func Stream() {
	anaconda.SetConsumerKey(Creds.Stream.ConsumerKey)
	anaconda.SetConsumerSecret(Creds.Stream.ConsumerSecret)
	api := anaconda.NewTwitterApi(Creds.Stream.AccessToken, Creds.Stream.AccessTokenSecret)
	if _, err := api.VerifyCredentials(); err != nil {
		log.Println("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens.")
		os.Exit(1)
	}

	user, err := api.GetUsersShow(User, url.Values{})
	if err != nil {
		log.Println(err)
	}

	tsv := Path + "/" + User + ".tsv"
	header := false
	if _, err = os.Stat(tsv); os.IsNotExist(err) {
		header = true
	}

	f, err := os.OpenFile(tsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	if header {
		if _, err = f.WriteString("event,created_at,screen_name,source,locaion,text\n"); err != nil {
			log.Fatal(err)
		}
	}

	stream := api.PublicStreamFilter(url.Values{
		"follow":       []string{user.IdStr},
		"filter_level": []string{"none"},
	})

	streamHandler := NewStreamHandler()
	streamHandler.Tweet = func(t anaconda.Tweet) {
		event := "TW"
		if t.RetweetedStatus != nil {
			event = "RT"
		}
		if t.QuotedStatus != nil {
			event = "QT"
		}
		if t.InReplyToUserIdStr != "" {
			event = "RP"
		}

		text := strings.Replace(t.Text, `"`, "", -1)
		createdAt, _ := time.Parse(time.RubyDate, t.CreatedAt)
		obj := &Obj{
			Event:      `"` + event + `"`,
			CreatedAt:  createdAt.Format("Mon Jan _2 15:04:05 +0000 2006"),
			ScreenName: `"` + t.User.ScreenName + `"`,
			Source:     `"` + regexp.MustCompile(`</?a(|\s+[^>]+)>`).ReplaceAllString(t.Source, "") + `"`,
			Location:   `"` + regexp.MustCompile(`(\r|\n|\t)`).ReplaceAllString(t.User.Location, " ") + `"`,
			Text:       `"` + regexp.MustCompile(`(\r|\n|\t)`).ReplaceAllString(text, " ") + `"`,
		}

		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", obj.Event, obj.CreatedAt, obj.ScreenName, obj.Source, obj.Location, obj.Text)
		if _, err = f.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n", obj.Event, obj.CreatedAt, obj.ScreenName, obj.Source, obj.Location, obj.Text)); err != nil {
			log.Fatal(err)
		}
	}

	go streamHandler.HandleChan(stream.C)

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println(<-ch)
}

// Post : Post a Tweet
func Post() {
	now := time.Now()

	tsv := Path + "/" + User + ".tsv"
	if _, err := os.Stat(tsv); os.IsNotExist(err) {
		log.Fatal(err)
		os.Exit(1)
	}
	f, _ := os.Open(tsv)
	defer f.Close()
	r := csv.NewReader(f)
	maxgroup := make(map[string]int)
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}

		createdAt, _ := time.Parse(time.RubyDate, line[1])
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.0 {
			maxgroup[line[2]]++
		}
	}
	max := 0
	top := ""
	for key, value := range maxgroup {
		if value >= max && strings.ToLower(key) != strings.ToLower(User) {
			max = maxgroup[key]
			top = key
		}
	}

	anaconda.SetConsumerKey(Creds.Stream.ConsumerKey)
	anaconda.SetConsumerSecret(Creds.Stream.ConsumerSecret)
	api := anaconda.NewTwitterApi(Creds.Stream.AccessToken, Creds.Stream.AccessTokenSecret)
	if _, err := api.VerifyCredentials(); err != nil {
		log.Println("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens.")
		os.Exit(1)
	}

	activity := User + "-activity-" + now.Format("2006-01-02") + ".png"
	buffactibity, _ := ioutil.ReadFile(Path + "/" + strings.ToLower(activity))
	actibitystring := base64.StdEncoding.EncodeToString(buffactibity)
	actibitymedia, _ := api.UploadMedia(actibitystring)

	mentions := User + "-mentions-" + now.Format("2006-01-02") + ".png"
	buffmentions, _ := ioutil.ReadFile(Path + "/" + strings.ToLower(mentions))
	mentionsstring := base64.StdEncoding.EncodeToString(buffmentions)
	mentionsmedia, _ := api.UploadMedia(mentionsstring)

	fmt.Println("Bot Spotted @" + top + " @" + User + " #BotOfTheDay #spam @TwitterSupport")

	meadiaIDStr := actibitymedia.MediaIDString + "," + mentionsmedia.MediaIDString
	api.PostTweet("Bot Spotted @"+top+" @"+User+" #BotOfTheDay #spam @TwitterSupport", url.Values{
		"media_ids": []string{meadiaIDStr},
	})
}

// List : List Various Metrics
func List() {
	tsv := Path + "/" + User + ".tsv"
	if _, err := os.Stat(tsv); os.IsNotExist(err) {
		log.Fatal(err)
		os.Exit(1)
	}
	f, _ := os.Open(tsv)
	defer f.Close()
	r := csv.NewReader(f)
	now := time.Now()
	maxgroup := make(map[string]int)
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}

		createdAt, _ := time.Parse(time.RubyDate, line[1])
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.0 {
			maxgroup[line[2]]++
		}
	}
	max := 0
	top := ""
	for key, value := range maxgroup {
		if value >= max && strings.ToLower(key) != strings.ToLower(User) {
			max = maxgroup[key]
			top = key
		}
	}
	GenData(top, false)
	GenData(User, true)
}

// Generate ...
func Generate() {
	cmd := exec.Command("python", "generate.py", "-u"+User)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// StreamHandler ...
type StreamHandler struct {
	All   func(m interface{})
	Tweet func(tweet anaconda.Tweet)
	Other func(m interface{})
}

// NewStreamHandler ...
func NewStreamHandler() *StreamHandler {
	return &StreamHandler{
		All:   func(m interface{}) {},
		Tweet: func(tweet anaconda.Tweet) {},
		Other: func(m interface{}) {},
	}
}

// Handle ...
func (d StreamHandler) Handle(m interface{}) {
	// d.All(m)

	switch t := m.(type) {
	case anaconda.Tweet:
		d.Tweet(t)
	default:
		d.Other(t)
	}
}

// HandleChan ...
func (d StreamHandler) HandleChan(c <-chan interface{}) {
	for m := range c {
		d.Handle(m)
	}
}

// GenData Generates Output Data
func GenData(u string, b bool) {
	now := time.Now()
	name := "target"
	if b {
		name = "source"
	}
	anaconda.SetConsumerKey(Creds.List.ConsumerKey)
	anaconda.SetConsumerSecret(Creds.List.ConsumerSecret)
	api := anaconda.NewTwitterApi(Creds.List.AccessToken, Creds.List.AccessTokenSecret)
	if _, err := api.VerifyCredentials(); err != nil {
		log.Println("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens.")
		os.Exit(1)
	}

	user, err := api.GetUsersShow(u, url.Values{})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	timeline, err := api.GetUserTimeline(url.Values{
		"screen_name":     []string{user.ScreenName},
		"count":           []string{"200"},
		"exclude_replies": []string{"false"},
		"include_rts":     []string{"true"},
	})

	if err != nil {
		log.Println(err)
	}
	type pair struct {
		i int
		k string
		v float64
	}
	d := []*pair{}
	var rts, rps, qts float64
	ment := make(map[string]float64)
	for i, t := range timeline {
		createdAt, _ := time.Parse(time.RubyDate, t.CreatedAt)
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.1 {
			if t.RetweetedStatus != nil {
				ment[t.RetweetedStatus.User.ScreenName]++
				rts++
			}
			if t.InReplyToScreenName != "" {
				ment[t.InReplyToScreenName]++
				rps++
			}
			if t.QuotedStatus != nil {
				ment[t.QuotedStatus.User.ScreenName]++
				qts++
			}
			idx := -1
			for i := range d {
				if d[i].k == createdAt.Add(3*time.Hour).Format("3 PM") {
					d[i].v++
					idx++
				}
			}
			if idx == -1 {
				d = append(d, &pair{
					k: createdAt.Add(3 * time.Hour).Format("3 PM"),
					v: 1,
					i: i,
				})
			}
		}
	}
	rev := []*pair{}
	for i := len(d) - 1; i >= 0; i-- {
		rev = append(rev, d[i])
	}

	activitytsv := Path + "/" + User + "-activity-" + name + "-" + now.Format("2006-01-02") + ".csv"
	os.Remove(activitytsv)
	f, err := os.OpenFile(activitytsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.WriteString("bot,created_at,count\n")
	for _, p := range rev {
		f.WriteString(fmt.Sprintf("%s,%s,%v\n", user.ScreenName, p.k, p.v))
	}

	mentionstsv := Path + "/" + User + "-mentions-" + name + "-" + now.Format("2006-01-02") + ".csv"
	os.Remove(mentionstsv)
	f, err = os.OpenFile(mentionstsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.WriteString("bot,user,count\n")
	for k, v := range ment {
		f.WriteString(fmt.Sprintf("%s,%s,%v\n", user.ScreenName, k, v))
	}
}
