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
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/ChimeraCoder/anaconda"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
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
		log.Println("You need to pass a command. Available Commands [list, stream, post].")
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
	tsv := Path + "/" + User + ".tsv"
	if _, err := os.Stat(tsv); os.IsNotExist(err) {
		log.Fatal(err)
		os.Exit(1)
	}
	f, _ := os.Open(tsv)
	defer f.Close()
	r := csv.NewReader(f)
	now := time.Now()
	// var todaysTweets []*Obj
	maxgroup := make(map[string]int)
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		createdAt, _ := time.Parse(time.RubyDate, line[1])
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.0 {
			// fmt.Println(line)
			/*
				todaysTweets = append(todaysTweets, &Obj{
					Event:      line[0],
					CreatedAt:  createdAt,
					ScreenName: line[2],
					Source:     line[3],
					Location:   line[4],
					Text:       line[5],
				})
			*/
			maxgroup[line[2]]++
		}
	}
	max := 0
	top := ""
	for key, value := range maxgroup {
		if value >= max {
			max = maxgroup[key]
			top = key
		}
	}

	fmt.Println(max, top)

	anaconda.SetConsumerKey(Creds.List.ConsumerKey)
	anaconda.SetConsumerSecret(Creds.List.ConsumerSecret)
	api := anaconda.NewTwitterApi(Creds.List.AccessToken, Creds.List.AccessTokenSecret)
	if _, err := api.VerifyCredentials(); err != nil {
		log.Println("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens.")
		os.Exit(1)
	}

	user, err := api.GetUsersShow(top, url.Values{})
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
	d := []*pair{} //make(map[string]float64)
	var rts, rps, qts float64
	ment := make(map[string]float64)
	for i, t := range timeline {
		createdAt, _ := time.Parse(time.RubyDate, t.CreatedAt)
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.3 {
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
				if d[i].k == createdAt.Add(3*time.Hour).Format("Mon Jan _2 15") {
					d[i].v++
					idx++
				}
			}
			if idx == -1 {
				d = append(d, &pair{
					k: createdAt.Add(3 * time.Hour).Format("Mon Jan _2 15"),
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
	Bars := []chart.Value{}
	for _, p := range rev {
		Bars = append(Bars, chart.Value{
			Value: p.v,
			Label: p.k + ":00",
			Style: chart.Style{
				StrokeWidth: 0.0,
				StrokeColor: drawing.Color{
					A: 0,
					B: 0,
					G: 0,
					R: 255,
				},
				FillColor: chart.ColorAlternateGray,
				Show:      true,
			},
		})
	}
	activityGraph := chart.BarChart{
		Width:  1440,
		Height: 720,
		Title:  "@" + user.ScreenName + " Activity",
		TitleStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
			Padding: chart.Box{
				Top:    48,
				Left:   48,
				Right:  48,
				Bottom: 48,
			},
		},
		XAxis: chart.Style{
			Show:                true,
			FontSize:            8.0,
			StrokeWidth:         0.5,
			TextHorizontalAlign: 1,
		},
		YAxis: chart.YAxis{
			NameStyle: chart.StyleShow(),
			AxisType:  chart.YAxisPrimary,
			Style: chart.Style{
				Show:        true,
				FontSize:    8.0,
				StrokeWidth: 0.5,
			},
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Bars: Bars,
	}

	af := top + "-" + now.Format("Mon-Jan-_2") + ".png"

	file, _ := os.Create(Path + "/" + "activity-" + strings.ToLower(af))
	err = activityGraph.Render(chart.PNG, file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	buffaf, _ := ioutil.ReadFile(Path + "/" + "activity-" + strings.ToLower(af))
	afstring := base64.StdEncoding.EncodeToString(buffaf)
	afmedia, _ := api.UploadMedia(afstring)

	actionsGraph := chart.PieChart{
		Width:  1440,
		Height: 720,
		Title:  "@" + user.ScreenName + " Actions",
		Background: chart.Style{
			Padding: chart.Box{
				Top:    48,
				Left:   48,
				Right:  48,
				Bottom: 48,
			},
		},
		TitleStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
		},
		SliceStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
			FontSize:            8.0,
			StrokeWidth:         2,
		},
		Values: []chart.Value{
			{Value: rts, Label: "Retweets (" + fmt.Sprintf("%v", rts) + "x)"},
			{Value: rps, Label: "Replies (" + fmt.Sprintf("%v", rps) + "x)"},
			{Value: qts, Label: "Quotes (" + fmt.Sprintf("%v", qts) + "x)"},
		},
	}

	file, _ = os.Create(Path + "/" + "actions-" + strings.ToLower(af))
	err = actionsGraph.Render(chart.PNG, file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	buffas, _ := ioutil.ReadFile(Path + "/" + "actions-" + strings.ToLower(af))
	asstring := base64.StdEncoding.EncodeToString(buffas)
	amedia, _ := api.UploadMedia(asstring)

	useMentionsValues := []chart.Value{}
	for k, v := range ment {
		if v > 1.0 {
			useMentionsValues = append(useMentionsValues, chart.Value{
				Value: v,
				Label: k + " (" + fmt.Sprintf("%v", v) + "x)",
			})
		}
	}
	userMentionsGraph := chart.PieChart{
		Width:  1440,
		Height: 720,
		Title:  "@" + user.ScreenName + " User Mentions (RTs. QTs, RPs)",
		Background: chart.Style{
			Padding: chart.Box{
				Top:    48,
				Left:   48,
				Right:  48,
				Bottom: 48,
			},
		},
		TitleStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
		},
		SliceStyle: chart.Style{
			Show:        true,
			FontSize:    8.0,
			StrokeWidth: 2,
			// StrokeColor: chart.ColorAlternateLightGray,
		},
		Values: useMentionsValues,
	}

	file, _ = os.Create(Path + "/" + "user-mentions-" + strings.ToLower(af))
	err = userMentionsGraph.Render(chart.PNG, file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	buffum, _ := ioutil.ReadFile(Path + "/" + "user-mentions-" + strings.ToLower(af))
	umstring := base64.StdEncoding.EncodeToString(buffum)
	ummedia, _ := api.UploadMedia(umstring)
	meadiaIDStr := afmedia.MediaIDString + "," + amedia.MediaIDString + "," + ummedia.MediaIDString
	post, _ := api.PostTweet("Bot Spotted @"+top+" @"+User+" #BotOfTheDay @twitter", url.Values{
		"media_ids": []string{meadiaIDStr},
	})

	fmt.Println(post)
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
	// var todaysTweets []*Obj
	maxgroup := make(map[string]int)
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		createdAt, _ := time.Parse(time.RubyDate, line[1])
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.0 {
			// fmt.Println(line)
			/*
				todaysTweets = append(todaysTweets, &Obj{
					Event:      line[0],
					CreatedAt:  createdAt,
					ScreenName: line[2],
					Source:     line[3],
					Location:   line[4],
					Text:       line[5],
				})
			*/
			maxgroup[line[2]]++
		}
	}
	max := 0
	top := ""
	for key, value := range maxgroup {
		if value >= max {
			max = maxgroup[key]
			top = key
		}
	}

	fmt.Println(max, top)

	anaconda.SetConsumerKey(Creds.List.ConsumerKey)
	anaconda.SetConsumerSecret(Creds.List.ConsumerSecret)
	api := anaconda.NewTwitterApi(Creds.List.AccessToken, Creds.List.AccessTokenSecret)
	if _, err := api.VerifyCredentials(); err != nil {
		log.Println("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens.")
		os.Exit(1)
	}

	user, err := api.GetUsersShow(top, url.Values{})
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
	d := []*pair{} //make(map[string]float64)
	var rts, rps, qts float64
	ment := make(map[string]float64)
	for i, t := range timeline {
		createdAt, _ := time.Parse(time.RubyDate, t.CreatedAt)
		diff := createdAt.Sub(now).Hours() / (24)
		if diff > -1.3 {
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
				if d[i].k == createdAt.Add(3*time.Hour).Format("Mon Jan _2 15") {
					d[i].v++
					idx++
				}
			}
			if idx == -1 {
				d = append(d, &pair{
					k: createdAt.Add(3 * time.Hour).Format("Mon Jan _2 15"),
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
	Bars := []chart.Value{}
	for _, p := range rev {
		Bars = append(Bars, chart.Value{
			Value: p.v,
			Label: p.k + ":00",
			Style: chart.Style{
				StrokeWidth: 0.0,
				StrokeColor: drawing.Color{
					A: 0,
					B: 0,
					G: 0,
					R: 255,
				},
				FillColor: chart.ColorAlternateGray,
				Show:      true,
			},
		})
	}
	activityGraph := chart.BarChart{
		Width:  1440,
		Height: 720,
		Title:  "@" + user.ScreenName + " Activity",
		TitleStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
			Padding: chart.Box{
				Top:    48,
				Left:   48,
				Right:  48,
				Bottom: 48,
			},
		},
		XAxis: chart.Style{
			Show:                true,
			FontSize:            8.0,
			StrokeWidth:         0.5,
			TextHorizontalAlign: 1,
		},
		YAxis: chart.YAxis{
			NameStyle: chart.StyleShow(),
			AxisType:  chart.YAxisPrimary,
			Style: chart.Style{
				Show:        true,
				FontSize:    8.0,
				StrokeWidth: 0.5,
			},
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Bars: Bars,
	}

	af := top + "-" + now.Format("Mon-Jan-_2") + ".png"

	file, _ := os.Create(Path + "/" + "activity-" + strings.ToLower(af))
	err = activityGraph.Render(chart.PNG, file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	actionsGraph := chart.PieChart{
		Width:  1440,
		Height: 720,
		Title:  "@" + user.ScreenName + " Actions",
		Background: chart.Style{
			Padding: chart.Box{
				Top:    48,
				Left:   48,
				Right:  48,
				Bottom: 48,
			},
		},
		TitleStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
		},
		SliceStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
			FontSize:            8.0,
			StrokeWidth:         2,
		},
		Values: []chart.Value{
			{Value: rts, Label: "Retweets (" + fmt.Sprintf("%v", rts) + "x)"},
			{Value: rps, Label: "Replies (" + fmt.Sprintf("%v", rps) + "x)"},
			{Value: qts, Label: "Quotes (" + fmt.Sprintf("%v", qts) + "x)"},
		},
	}

	file, _ = os.Create(Path + "/" + "actions-" + strings.ToLower(af))
	err = actionsGraph.Render(chart.PNG, file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	useMentionsValues := []chart.Value{}
	for k, v := range ment {
		if v > 1.0 {
			useMentionsValues = append(useMentionsValues, chart.Value{
				Value: v,
				Label: k + " (" + fmt.Sprintf("%v", v) + "x)",
			})
		}
	}
	userMentionsGraph := chart.PieChart{
		Width:  1440,
		Height: 720,
		Title:  "@" + user.ScreenName + " User Mentions (RTs. QTs, RPs)",
		Background: chart.Style{
			Padding: chart.Box{
				Top:    48,
				Left:   48,
				Right:  48,
				Bottom: 48,
			},
		},
		TitleStyle: chart.Style{
			Show:                true,
			TextHorizontalAlign: 1,
		},
		SliceStyle: chart.Style{
			Show:        true,
			FontSize:    8.0,
			StrokeWidth: 2,
			// StrokeColor: chart.ColorAlternateLightGray,
		},
		Values: useMentionsValues,
	}

	file, _ = os.Create(Path + "/" + "user-mentions-" + strings.ToLower(af))
	err = userMentionsGraph.Render(chart.PNG, file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
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
