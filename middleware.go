package ex

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

type Middleware func(next HandlerFunc) HandlerFunc

// 로그처리
// 로그를 남기는 미들웨어
func logHandler(next HandlerFunc) HandlerFunc {
	return func(c *Context) {
		t := time.Now()
		next(c)
		log.Printf("[%s] %q %v\n", c.Request.Method, c.Request.URL.String(), time.Now().Sub(t))
	}
}

// 에러처리
// 핸들러 내에서 패닉이 발생했을 때 웹서버를 종료하지 않고 500 에러를 출력하는 recoverHandler
func recoverHandler(next HandlerFunc) HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				http.Error(c.ResponseWriter, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next(c)
	}
}

// 웹 요청 내용 파싱 미들웨어
// POST에 전송된 Form 데이터를 Context의 Params에 담는다.
func parseFormHandler(next HandlerFunc) HandlerFunc {
	return func(c *Context) {
		c.Request.ParseForm()
		fmt.Println(c.Request.PostForm)
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				c.Params[k] = v[0]
			}
		}
		next(c)
	}
}

// JSON 데이터를 해석해서 Context의 Params에 담는다.
func parseJsonBodyHandler(next HandlerFunc) HandlerFunc {
	return func(c *Context) {
		var m map[string]interface{}
		if json.NewDecoder(c.Request.Body).Decode(&m); len(m) > 0 {
			for k, v := range m {
				c.Params[k] = v
			}
		}
		next(c)
	}
}

// 정적 파일 내용을 전달하는 미들웨어
func staticHandler(next HandlerFunc) HandlerFunc {
	var (
		dir       = http.Dir(".")
		indexFile = "index.html"
	)
	return func(c *Context) {
		// http 메서드가 GET이나 HEAD 가 아니면 바로 다음 핸들러 수행
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			next(c)
			return
		}
		file := c.Request.URL.Path
		// URL 경로에 해당하는 파일 열기 시도
		f, err := dir.Open(file)
		if err != nil {
			// URL 경로에 해당하는 파일 열기에 실패하면 바로 다음 핸들러 수행
			next(c)
			return
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			// 파일의 상태가 비정상이면 바로 다음 핸들러 수행
			next(c)
			return
		}

		// URL 경로가 디렉터리면 indexFile을 사용
		if fi.IsDir() {
			// 디렉터리 경로를 URL로 사용하면 경로 끝에 "/" 를 붙여야 함
			if !strings.HasSuffix(c.Request.URL.Path, "/") {
				http.Redirect(c.ResponseWriter, c.Request, c.Request.URL.Path+"/", http.StatusFound)
				return
			}

			// 디렉터리를 가리키는 URL 경로에 indexFile 이름을 붙여서 전체 파일 경로 생성
			file = path.Join(file, indexFile)

			// indexFile 열기 시도
			f, err = dir.Open(file)
			if err != nil {
				next(c)
				return
			}
			defer f.Close()

			fi, err = f.Stat()
			if err != nil || fi.IsDir() {
				// indexFile 상태가 정상이 아니면 바로 다음 핸들러 수행
				next(c)
				return
			}
		}
		// file 의 내용 전달(next 핸들러로 제어권을 넘기지 않고 요청 처리를 종료함)
		http.ServeContent(c.ResponseWriter, c.Request, file, fi.ModTime(), f)

	}
}
