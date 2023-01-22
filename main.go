package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gocolly/colly/v2"
)

func postRequest(url string, data map[string]string, cb func(r *http.Response)) {
	jsonPayload, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("cannot parse data: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Fatalf("failed to post request to %s : %v", url, err)
	}

	cb(resp)
}

func login(c *colly.Collector, username, password string, logintoken string) {
	c.Post("https://kursusvmlepkom.gunadarma.ac.id/login/index.php", map[string]string{
		"username":   username,
		"password":   password,
		"logintoken": logintoken,
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Login success", r.URL)
	})
}

func getLoginToken(c *colly.Collector) string {
	var loginToken string

	c.OnHTML("input[name]", func(h *colly.HTMLElement) {
		if h.Attr("name") == "logintoken" {
			value := h.Attr("value")
			loginToken = value
		}
	})

	c.Visit("https://kursusvmlepkom.gunadarma.ac.id/login/index.php")

	return loginToken
}

func getCourseNames(c *colly.Collector) []string {
	var courseNames []string
	c.OnHTML("h3.coursename", func(h *colly.HTMLElement) {
		a := h.ChildText("a")
		if !strings.HasPrefix(a, "ACTIVITY") {
			href := h.ChildAttr("a", "href")
			courseNames = append(courseNames, href)
		}
	})

	c.Visit("https://kursusvmlepkom.gunadarma.ac.id")

	if len(courseNames) < 1 {
		return nil
	}

	return courseNames
}

type Participant struct {
	Name       string
	NPM        string
	Class      string
	LastAccess string
	Grade      Grade
	ExamGrade  ExamGrade
}

type Grade struct {
	Pert1 string
	Pert2 string
	Pert3 string
	Pert4 string
	Pert5 string
	Pert6 string
	Pert7 string
	Pert8 string
}

type ExamGrade struct {
	Ujian1 string
	Ujian2 string
	Ujian3 string
}

func getParticipants(c *colly.Collector, courseUrl string) []string {
	urlString, err := url.Parse(courseUrl)
	if err != nil {
		fmt.Println(err)
	}

	id := urlString.Query().Get("id")
	if id == "" {
		fmt.Println("No id")
	}

	// var students []Students
	c.OnHTML("th.cell.c1 > a", func(h *colly.HTMLElement) {
		if h.Text == "" {
			return
		}

		stringSplit := strings.Split(h.Text, " ")

		npm := stringSplit[len(stringSplit)-1]
		class := stringSplit[len(stringSplit)-2]
		name := strings.Join(stringSplit[:len(stringSplit)-2], " ")

		assistantRoles := []string{"Asisten", "PJ"}
		for _, role := range assistantRoles {
			if npm == role {
				return
			}
		}

		participant := Participant{
			Name:  name,
			NPM:   npm,
			Class: class,
		}
		fmt.Println(participant)
	})

	participantURL := fmt.Sprintf("https://kursusvmlepkom.gunadarma.ac.id/user/index.php?id=%s&perpage=5000", id)
	c.Visit(participantURL)

	return nil

}

func main() {
	c := colly.NewCollector()

	loginToken := getLoginToken(c)

	login(c, "username", "password", loginToken)

	courseNames := getCourseNames(c)

	getParticipants(c, courseNames[0])

	c.OnRequest(func(r *colly.Request) {
		log.Println(r.URL)

	})

	c.Visit("https://kursusvmlepkom.gunadarma.ac.id")
}
