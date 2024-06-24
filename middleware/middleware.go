package middleware

import (
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"pizza-factory-go/response"
	"time"
)

type Middleware func(http.Handler) http.Handler

func CreateStack(xs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(xs) - 1; i >= 0; i-- {
			next = xs[i](next)
		}

		return next

	}
}

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) Test(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func Logging(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(wrapped, r)
		//log.Println(wrapped.statusCode, r.Method, r.URL.Path, time.Since(start))
		log.Println(wrapped.statusCode, r.Method, r.URL.Path, time.Since(start))
	})

}

func AuthHeaderRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Перебор всех заголовков
		authHeader := r.Header.Get("X-Auth-Key")
		if authHeader == "" {
			response.WritePlainText(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		hashedPassword := os.Getenv("APP_AUTH_HEADER_BCRYPT")

		//hashed password is used because of possible timing attack (https://en.wikipedia.org/wiki/Timing_attack)
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(authHeader)); err != nil {
			response.WritePlainText(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
