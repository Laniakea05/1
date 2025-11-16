package handlers

import "github.com/gin-gonic/gin"

func IndexPage(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{})
}

func LoginPage(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{})
}

func RegisterPage(c *gin.Context) {
	c.HTML(200, "register.html", gin.H{})
}

func DashboardPage(c *gin.Context) {
	c.HTML(200, "dashboard.html", gin.H{})
}

func TestsPage(c *gin.Context) {
	c.HTML(200, "tests.html", gin.H{})
}

func TestTakingPage(c *gin.Context) {
	c.HTML(200, "test-taking.html", gin.H{})
}

func TestResultPage(c *gin.Context) {
	c.HTML(200, "test-result.html", gin.H{})
}

func AdminPage(c *gin.Context) {
	c.HTML(200, "admin.html", gin.H{})
}