package routes

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/peyzor/live-broadcast-chat/broadcast"
	cMiddleware "github.com/peyzor/live-broadcast-chat/routes/middleware"
)

var bc *broadcast.Broadcast

type TemplateRegistry struct {
	templates map[string]*template.Template
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data any, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		err := errors.New("Template not found -> " + name)
		return err
	}
	// if we are loading a partial base will be missing
	base := tmpl.Lookup("base.html")
	if base == nil {
		return tmpl.ExecuteTemplate(w, name, data)
	}
	return tmpl.ExecuteTemplate(w, "base.html", data)

}

func Setup(e *echo.Echo) error {
	if bc == nil {
		bc = broadcast.NewBroadcast()
	}
	SetupStaticAssets(e)

	templates := make(map[string]*template.Template)
	templates["home.html"] = template.Must(
		template.New("").ParseFiles(
			"templates/pages/home.html",
			"templates/base.html",
		),
	)
	templates["404.html"] = template.Must(
		template.New("").ParseFiles(
			"templates/pages/404.html",
			"templates/base.html",
		),
	)
	templates["live_chat.html"] = template.Must(
		template.New("").ParseFiles(
			"templates/pages/live_chat.html",
			"templates/base.html",
			"templates/partials/chat_input.html",
		),
	)
	templates["about.html"] = template.Must(
		template.New("").ParseFiles("templates/pages/about.html", "templates/base.html"),
	)
	templates["clicked.html"] = template.Must(
		template.New("").ParseFiles("templates/partials/clicked.html"),
	)
	templates["chat_msg.html"] = template.Must(
		template.New("").ParseFiles("templates/partials/chat_msg.html"),
	)
	templates["chat_input.html"] = template.Must(
		template.New("").ParseFiles("templates/partials/chat_input.html"),
	)

	e.Renderer = &TemplateRegistry{
		templates: templates,
	}

	root := e.Group("/", cMiddleware.CacheControl(0))
	root.GET("", func(c echo.Context) error {
		// Perform the redirect
		return c.Redirect(http.StatusFound, "/live-chat")
	})

	root.GET("404", func(c echo.Context) error {
		return c.Render(http.StatusOK, "404.html", map[string]any{})
	})

	root.GET("about", func(c echo.Context) error {
		return c.Render(http.StatusOK, "about.html", map[string]any{})
	})

	root.GET("chatroom", func(c echo.Context) error {
		handler := handleSSE(c, e.Renderer)
		handler(c.Response().Writer, c.Request())
		return nil
	})

	root.GET("live-chat", func(c echo.Context) error {
		return c.Render(http.StatusOK, "live_chat.html", map[string]any{})
	})

	e.POST("sendChat", func(c echo.Context) error {
		msg := c.FormValue("msg")

		if bc != nil && msg != "" {
			errs := bc.Send(msg)
			for id, err := range errs {
				e.Logger.Errorf("listener: %s %s", id, err)
			}
		}
		return c.Render(http.StatusOK, "chat_input.html", map[string]any{})
	})

	return nil
}

func handleSSE(c echo.Context, t echo.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, _ := w.(http.Flusher)

		list := bc.AddListener()
		defer bc.RemoveListener(list)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg := <-list.Chan:
				err := t.Render(w, "chat_msg.html", map[string]any{
					"msg": msg,
				}, c)
				if err != nil {
					fmt.Println(err)
				}
				_, err = fmt.Fprintf(w, "\n\n")
				if err != nil {
					fmt.Println(err)
				}
				flusher.Flush()
			case <-ticker.C:
				_, err := fmt.Fprintf(w, "keepalive: \n\n")
				if err != nil {
					fmt.Println(err)
				}
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

func SetupStaticAssets(e *echo.Echo) {
	e.Use(cMiddleware.CacheControl(0), middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   "static",
		Browse: false,
	}))
}
