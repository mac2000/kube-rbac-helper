package main

// go get -u k8s.io/client-go@latest
// go get -u github.com/rs/zerolog/log
import (
	"embed"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

//go:embed resources
var resources embed.FS

//go:embed templates
var templates embed.FS

//go:embed resources/index.html
var index []byte

var client *kubernetes.Clientset
var err error

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	client, err = getClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to initialize Kubernetes client")
	}

	tpl, err := template.ParseFS(templates, "templates/index.html.tmpl")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse template")
	}

	http.Handle("/resources/", http.FileServer(http.FS(resources)))

	http.HandleFunc("/left", func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw.Header().Add("Content-Type", "text/html")

		data, err := getTree()
		tpl.Execute(rw, struct {
			Data  map[string]map[string][]string
			Error error
		}{Data: data, Error: err})

		if err != nil {
			log.Warn().Err(err).Str("path", r.URL.Path).Dur("took", time.Duration(time.Since(start))).Msg("served home endpoint")
		} else {
			log.Info().Str("path", r.URL.Path).Dur("took", time.Duration(time.Since(start))).Msg("served home endpoint")
		}
	})

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw.Header().Add("Content-Type", "text/html")
		rw.Write(index)
		log.Info().Str("path", r.URL.Path).Dur("took", time.Duration(time.Since(start))).Msg("served home endpoint")
	})

	log.Info().Msg("Starting server on 0.0.0.0:8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to start server")
	}
}

func contains(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

func getTree() (map[string]map[string][]string, error) {
	tree := map[string]map[string][]string{}
	groups, err := client.DiscoveryClient.ServerResources()
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		var groupName string
		idx := strings.Index(group.GroupVersion, "/")
		if idx > 0 {
			groupName = group.GroupVersion[:idx]
		} else {
			groupName = ""
		}
		for _, resource := range group.APIResources {
			if tree[groupName] == nil {
				tree[groupName] = map[string][]string{}
			}
			if tree[groupName][resource.Name] == nil {
				tree[groupName][resource.Name] = []string{}
			}
			for _, verb := range resource.Verbs {
				// same resource might contain multiple version and each will have same verbs, but we are removing versions, so ended up with duplicated verbs
				if !contains(tree[groupName][resource.Name], verb) {
					tree[groupName][resource.Name] = append(tree[groupName][resource.Name], verb)
				}
			}
		}
	}
	return tree, nil
}

func getConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Info().Str("host", config.Host).Msg("in cluster config")
		return config, nil
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		config, err = clientcmd.BuildConfigFromFlags("", env)
		if err == nil {
			log.Info().Str("host", config.Host).Msg("kubeconfig environment variable")
			return config, nil
		}
	}
	config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		return nil, err
	}
	log.Info().Str("host", config.Host).Msg("~/.kube/config variable")
	return config, nil
}

func getClient() (*kubernetes.Clientset, error) {
	// cfg := filepath.Join(homedir.HomeDir(), ".kube", "config")
	// cfg := "/Users/mac/Documents/dotfiles/kube/cub.yml"
	// config, err := clientcmd.BuildConfigFromFlags("", cfg)
	// if err != nil {
	// 	return nil, err
	// }

	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
