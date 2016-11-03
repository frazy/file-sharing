package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"time"
)

type fileType struct {
	Name    string
	ModTime string
	Size    string
	Path    string
	IsDir   string
}

type filesType []fileType

func (files filesType) Len() int { return len(files) }
func (files filesType) Less(i, j int) bool {
	return (files[i].IsDir + files[i].Name) < (files[j].IsDir + files[j].Name)
}
func (files filesType) Swap(i, j int) { files[i], files[j] = files[j], files[i] }

type fileHandler struct {
}

func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestIp := getRequestIp(r)
	log.Printf("[%s] >> %s %s", requestIp, r.Method, r.URL.String())
	status := http.StatusOK
	defer func() {
		log.Printf("[%s] << %d %s", requestIp, status, time.Since(start))
	}()

	root := http.Dir(rootDir)
	dir := path.Clean("/" + r.URL.Path)
	f, err := root.Open(dir)
	if err != nil {
		log.Printf("Open %s error: %v", dir, err)
		if os.IsNotExist(err) {
			status = http.StatusNotFound
			http.Error(w, "404", http.StatusNotFound)
			return
		}
		status = http.StatusInternalServerError
		http.Error(w, "500", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Printf("Stat %s error: %v", dir, err)
		status = http.StatusInternalServerError
		http.Error(w, "500", http.StatusInternalServerError)
		return
	}

	if fi.IsDir() {
		fis, err := f.Readdir(-1)
		if err == io.EOF {
			// no sub files or dirs
		} else {
			check(err)
		}

		files := make(filesType, 0)
		for _, fi := range fis {
			// log.Printf("fi=%v", fi)
			file := fileType{}
			file.Name = fi.Name()
			file.ModTime = fi.ModTime().Format("2006-01-02 15:04")
			file.Path = path.Clean(dir + "/" + file.Name)
			if fi.IsDir() {
				file.Size = "-"
				file.Path += "/"
				file.IsDir = "D"
			} else {
				file.Size = formatSize(fi.Size())
				file.Path += "?dl"
				file.IsDir = "F"
			}
			files = append(files, file)
		}
		sort.Sort(files)

		data := make(map[string]interface{})
		data["title"] = dir
		data["files"] = files

		t, err := template.ParseFiles(template_dir + "index.html")
		check(err)
		err = t.Execute(w, data)
		check(err)
		return
	} else {
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
		return
	}
}

const (
	template_dir = "./view/"
)

var (
	listen  string
	rootDir string
)

func init() {
	flag.StringVar(&listen, "l", ":80", "listen")
	flag.StringVar(&rootDir, "d", "./", "file root")
	flag.Parse()
}

func main() {
	server := http.Server{
		Addr:        listen,
		Handler:     &fileHandler{},
		ReadTimeout: 10 * time.Second,
	}

	log.Printf("Start server %s", server.Addr)
	log.Fatalln(server.ListenAndServe())
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func formatSize(i int64) string {
	const (
		wt = 1024
	)

	size := float64(i)
	if size < wt {
		return strconv.FormatFloat(size, 'f', 2, 64) + ""
	}

	size /= wt
	if size < wt {
		return strconv.FormatFloat(size, 'f', 2, 64) + "K"
	}

	size /= wt
	if size < wt {
		return strconv.FormatFloat(size, 'f', 2, 64) + "M"
	}

	size /= wt
	return strconv.FormatFloat(size, 'f', 2, 64) + "G"
}

func getRequestIp(req *http.Request) (ip string) {
	ip = req.Header.Get("X-Real-IP")
	if ip == "" {
		ip = req.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = req.RemoteAddr
		}
	}
	return
}
