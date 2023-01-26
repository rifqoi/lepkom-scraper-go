package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
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

func login(c *colly.Collector, username, password string, logintoken string) {
	loginURL := "https://kursusvmlepkom.gunadarma.ac.id/login/index.php"
	c.Post(loginURL, map[string]string{
		"username":   username,
		"password":   password,
		"logintoken": logintoken,
	})
}

func checkLogin(c *colly.Collector) bool {
	userProfileLogin := "https://kursusvmlepkom.gunadarma.ac.id/user/profile.php"

	authenticated := true

	// This is just to check if we're authenticated by checking the logintoken
	c.OnHTML("input[name]", func(h *colly.HTMLElement) {
		if h.Attr("name") == "logintoken" {
			value := h.Attr("value")
			fmt.Println(value)
			if len(value) > 0 {
				authenticated = false
			}
		}
	})

	c.Visit(userProfileLogin)

	return authenticated
}

type Course struct {
	CourseName string
	CourseURL  string
	CourseID   string
}

func parseCourseID(courseURL string) string {
	urlString, err := url.Parse(courseURL)
	if err != nil {
		return ""
	}

	id := urlString.Query().Get("id")
	if id == "" {
		return ""
	}

	return id
}

func getCourses(c *colly.Collector) ([]Course, error) {
	var courses []Course
	fmt.Println(c)
	c.OnHTML("h3.coursename", func(h *colly.HTMLElement) {
		a := h.ChildText("a")
		a = strings.ToUpper(a)
		if !strings.HasPrefix(a, "ACTIVITY") && !strings.Contains(a, "TESTING") {
			href := h.ChildAttr("a", "href")

			courseID := parseCourseID(href)
			if courseID == "" {
				return
			}

			courses = append(courses, Course{
				CourseName: strings.TrimSpace(a),
				CourseURL:  href,
				CourseID:   courseID,
			})
		}
	})

	c.Visit("https://kursusvmlepkom.gunadarma.ac.id")

	if len(courses) < 1 {
		return nil, fmt.Errorf("No courses found")
	}

	return courses, nil
}

type Participant struct {
	Name       string
	NPM        string
	Class      string
	LastAccess string
	Pertemuan  []Pertemuan
	Ujian      []Ujian
	Delete     bool
	DeleteAt   []int
}

func (p *Participant) JumlahDelete() int {
	return len(p.DeleteAt)
}

func (p *Participant) IsDelete() bool {
	for _, pert := range p.Pertemuan {
		hadir := pert.IsHadir()
		if hadir {
			continue
		}

		p.DeleteAt = append(p.DeleteAt, pert.PertemuanKe)

		if p.JumlahDelete() == 2 {
			p.Delete = true
			break
		}
	}

	return p.Delete
}

type Pertemuan struct {
	PertemuanKe int
	Hadir       bool
	PreTest     Tes
	PostTest    Tes
}

func (m *Pertemuan) IsHadir() bool {
	if m.PreTest.Mengerjakan || m.PostTest.Mengerjakan {
		m.Hadir = true
	} else {
		m.Hadir = false
	}

	return m.Hadir
}

type Ujian struct {
	UjianKe int
	Test    Tes
}

type Tes struct {
	Grade       int
	Mengerjakan bool
}

func getParticipants(c *colly.Collector, course Course) []string {
	c.OnHTML("th.cell.c0 > a.username", func(h *colly.HTMLElement) {
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

	gradeURL := fmt.Sprintf("https://kursusvmlepkom.gunadarma.ac.id/grade/report/grader/index.php?id=%s", course.CourseID)
	c.Visit(gradeURL)

	return nil
}

func main() {
	godotenv.Load()
	c := colly.NewCollector()

	loginToken := getLoginToken(c)

	login(c, os.Getenv("username"), os.Getenv("password"), loginToken)
	ok := checkLogin(c.Clone())
	if !ok {
		log.Fatal("Username or password is incorrect.")
	}

	courseCollector := c.Clone()
	// participantCol := c.Clone()

	courses, _ := getCourses(courseCollector)
	fmt.Println(courses)

	for _, course := range courses {
		getParticipants(c, course)
	}

	c.OnRequest(func(r *colly.Request) {
		log.Println(r.URL)

	})

	c.Visit("https://kursusvmlepkom.gunadarma.ac.id")
}
