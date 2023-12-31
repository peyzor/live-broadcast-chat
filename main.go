package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/peyzor/live-broadcast-chat/routes"
)

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  log.ERROR,
	}))

	e.HTTPErrorHandler = customHTTPErrorHandler
	err := routes.Setup(e)
	if err != nil {
		panic(fmt.Sprintln("failed to setup routes: ", err))
	}

	err = e.Start(":8080")
	if err != nil {
		e.Logger.Warn(err)
	}
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	host := c.Request().Host
	URI := c.Request().RequestURI
	qs := c.QueryString()

	c.Logger().Error(err, fmt.Sprintf(" on: %s%s%s error code: %d", host, URI, qs, code))
	if code == http.StatusNotFound {
		c.Redirect(http.StatusTemporaryRedirect, "/404")
	}
	c.String(code, fmt.Sprintf("error code: %d", code))
}
