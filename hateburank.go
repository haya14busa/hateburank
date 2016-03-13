package hateburank

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"golang.org/x/net/context"

	// http://y-anz-m.blogspot.jp/2015/09/google-app-engine-go-google-cloud.html
	newappengine "google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"

	// Import appengine urlfetch package, that is needed to make http call to the api
	"appengine"
	"appengine/urlfetch"

	"github.com/ChimeraCoder/anaconda"
	"github.com/haya14busa/hatebu-ranking-entries/category"
	hateburankurl "github.com/haya14busa/hatebu-ranking-entries/url"
)

const debug = false

var api *anaconda.TwitterApi

var (
	dailyCategories = []category.Category{
		category.Hotentry,
		category.It,
		category.Game,
	}

	weeklyCategories = []category.Category{
		category.Hotentry,
		category.It,
		category.Game,
	}

	monthlyCategories = []category.Category{
		category.Hotentry,
		category.It,
		category.Game,
	}
)

func init() {
	anaconda.SetConsumerKey(Consumer_Key)
	anaconda.SetConsumerSecret(Consumer_Secret)
	api = anaconda.NewTwitterApi(Access_Token, Access_Token_Secret)

	http.HandleFunc("/api/tweet/daily", dailyHandler)
	http.HandleFunc("/api/tweet/weekly", weeklyHandler)
	http.HandleFunc("/api/tweet/monthly", monthlyHandler)
	http.HandleFunc("/", topHandler)
}

func setContext(r *http.Request) {
	ctx := appengine.NewContext(r)
	api.HttpClient.Transport = &urlfetch.Transport{Context: ctx}
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello from hateburank!")
}

type Tweeted struct {
	Url string
}

func dailyHandler(w http.ResponseWriter, r *http.Request) {
	setContext(r)
	ctx := newappengine.NewContext(r)

	force := r.URL.Query().Get("force") != ""

	var errors []string

	for _, c := range dailyCategories {
		if err := daily(ctx, c, force); err != nil {
			log.Errorf(ctx, "Fail to run daily job for %v: %v", c, err.Error())
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		http.Error(w, strings.Join(errors, "\n"), 500)
		return
	}

	fmt.Fprint(w, fmt.Sprintf("Succeed to run daily job"))
}

func weeklyHandler(w http.ResponseWriter, r *http.Request) {
	setContext(r)
	ctx := newappengine.NewContext(r)

	force := r.URL.Query().Get("force") != ""

	var errors []string

	for _, c := range weeklyCategories {
		if err := weekly(ctx, c, force); err != nil {
			log.Errorf(ctx, "Fail to run weekly job for %v: %v", c, err.Error())
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		http.Error(w, strings.Join(errors, "\n"), 500)
		return
	}

	fmt.Fprint(w, fmt.Sprintf("Succeed to run weekly job"))
}

func monthlyHandler(w http.ResponseWriter, r *http.Request) {
	setContext(r)
	ctx := newappengine.NewContext(r)

	force := r.URL.Query().Get("force") != ""

	var errors []string

	for _, c := range monthlyCategories {
		if err := monthly(ctx, c, force); err != nil {
			log.Errorf(ctx, "Fail to run monthly job for %v: %v", c, err.Error())
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		http.Error(w, strings.Join(errors, "\n"), 500)
		return
	}

	fmt.Fprint(w, fmt.Sprintf("Succeed to run monthly job"))
}

func daily(ctx context.Context, c category.Category, force bool) error {
	dailyurl := hateburankurl.DailyFromCategoryLatest(c)
	key := datastore.NewKey(ctx, "dailyurl", dailyurl, 0, nil)

	if didTweet(ctx, key, dailyurl) && !force {
		return fmt.Errorf("Already tweeted: %s", dailyurl)
	}

	if _, err := tweetDaily(ctx, c, dailyurl); err != nil {
		return err
	}
	log.Debugf(ctx, "Succeed to post tweet for url: %v", dailyurl)

	if err := saveTweet(ctx, key, dailyurl); err != nil {
		return err
	}
	log.Debugf(ctx, "Succeed to save tweet for url: %v", dailyurl)
	return nil
}

func weekly(ctx context.Context, c category.Category, force bool) error {
	weeklyurl := hateburankurl.WeeklyFromCategoryLatest(c)
	key := datastore.NewKey(ctx, "weeklyurl", weeklyurl, 0, nil)

	if didTweet(ctx, key, weeklyurl) && !force {
		return fmt.Errorf("Already tweeted: %s", weeklyurl)
	}

	if _, err := tweetWeekly(ctx, c, weeklyurl); err != nil {
		return err
	}
	log.Debugf(ctx, "Succeed to post tweet for url: %v", weeklyurl)

	if err := saveTweet(ctx, key, weeklyurl); err != nil {
		return err
	}
	log.Debugf(ctx, "Succeed to save tweet for url: %v", weeklyurl)
	return nil
}

func monthly(ctx context.Context, c category.Category, force bool) error {
	monthlyurl := hateburankurl.MonthlyFromCategoryLatest(c)
	key := datastore.NewKey(ctx, "monthlyurl", monthlyurl, 0, nil)

	if didTweet(ctx, key, monthlyurl) && !force {
		return fmt.Errorf("Already tweeted: %s", monthlyurl)
	}

	if _, err := tweetMonthly(ctx, c, monthlyurl); err != nil {
		return err
	}
	log.Debugf(ctx, "Succeed to post tweet for url: %v", monthlyurl)

	if err := saveTweet(ctx, key, monthlyurl); err != nil {
		return err
	}
	log.Debugf(ctx, "Succeed to save tweet for url: %v", monthlyurl)
	return nil
}

func didTweet(ctx context.Context, key *datastore.Key, url string) bool {
	if debug {
		return false
	}
	tweeted := new(Tweeted)
	if err := datastore.Get(ctx, key, tweeted); err != nil {
		return false
	}
	return true
}

func saveTweet(ctx context.Context, key *datastore.Key, url string) error {
	tweeted := new(Tweeted)
	tweeted.Url = url
	if _, err := datastore.Put(ctx, key, tweeted); err != nil {
		return err
	}
	return nil
}

const daily_template = `[hateburank-daily:{{.Category}}] {{.StartDate}} の日間はてなブックマークランキング {{.Url}}`

func tweetDaily(ctx context.Context, c category.Category, url string) (anaconda.Tweet, error) {
	startDate := time.Now().AddDate(0, 0, -1)

	tmpl := template.Must(template.New("daily_template").Parse(daily_template))
	var msg bytes.Buffer
	data := struct {
		Category  string
		StartDate string
		Url       string
	}{
		Category:  c.String(),
		StartDate: startDate.Format("2006/01/02"),
		Url:       url,
	}
	if err := tmpl.Execute(&msg, data); err != nil {
		return anaconda.Tweet{}, err
	}
	return tweet(ctx, msg.String())
}

const weekly_template = `[hateburank-weekly:{{.Category}}] {{.StartDate}}-{{.EndDate}} の週間はてなブックマークランキング {{.Url}}`

func tweetWeekly(ctx context.Context, c category.Category, url string) (anaconda.Tweet, error) {
	startDate := toMonday(time.Now().AddDate(0, 0, -7))
	endDate := startDate.AddDate(0, 0, 6)

	tmpl := template.Must(template.New("weekly_template").Parse(weekly_template))
	var msg bytes.Buffer
	data := struct {
		Category  string
		StartDate string
		EndDate   string
		Url       string
	}{
		Category:  c.String(),
		StartDate: startDate.Format("2006/01/02"),
		EndDate:   endDate.Format("2006/01/02"),
		Url:       url,
	}
	if err := tmpl.Execute(&msg, data); err != nil {
		return anaconda.Tweet{}, err
	}
	return tweet(ctx, msg.String())
}

const monthly_template = `[hateburank-monthly:{{.Category}}] {{.StartDate}} の月間はてなブックマークランキング {{.Url}}`

func tweetMonthly(ctx context.Context, c category.Category, url string) (anaconda.Tweet, error) {
	startDate := time.Now().AddDate(0, -1, 0)

	tmpl := template.Must(template.New("monthly_template").Parse(monthly_template))
	var msg bytes.Buffer
	data := struct {
		Category  string
		StartDate string
		Url       string
	}{
		Category:  c.String(),
		StartDate: startDate.Format("2006年01月"),
		Url:       url,
	}
	if err := tmpl.Execute(&msg, data); err != nil {
		return anaconda.Tweet{}, err
	}
	return tweet(ctx, msg.String())
}

func tweet(ctx context.Context, message string) (anaconda.Tweet, error) {
	if debug {
		log.Debugf(ctx, "pseudo-tweet: %s", message)
		return anaconda.Tweet{}, nil
	}
	return api.PostTweet(message, url.Values{})
}

func toMonday(date time.Time) time.Time {
	return date.AddDate(0, 0, int(-date.Weekday()+1))
}
