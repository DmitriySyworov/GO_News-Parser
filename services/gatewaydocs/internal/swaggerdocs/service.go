package swaggerdocs

import (
	docsconfig "app/gatewaydocs/config"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"shared/loggers"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ServiceSwaggerDocs struct {
	DocsMap     sync.Map
	Logger      *loggers.Logger
	Conf        *docsconfig.Config
	ServiceList []ListDocs
}

type ListDocs struct {
	Service string
	Url     string
}

const (
	AuthService   = "auth"
	BudgetService = "budget"
)

func NewServiceSwaggerDocs(logger *loggers.Logger, conf *docsconfig.Config) *ServiceSwaggerDocs {
	return &ServiceSwaggerDocs{
		Logger: logger,
		Conf:   conf,
		ServiceList: []ListDocs{
			{Service: AuthService, Url: conf.AuthUserURL + "/swagger/doc.json"},
		},
	}
}

type InfoServices struct {
	Name string
	Url  string
}

func (s *ServiceSwaggerDocs) GetDocs() []InfoServices {
	listInfoServices := []InfoServices{
		{Name: "Auth Service API", Url: "/docs/api?service=auth"},
		{Name: "All API", Url: "/docs/api"},
	}
	return listInfoServices
}

func (s *ServiceSwaggerDocs) GetDocsAPI(service string) ([]byte, error) {
	if service != "" {
		docsData, found := s.DocsMap.Load(service)
		if !found {
			return nil, ErrNotFoundDocs
		}
		return docsData.([]byte), nil
	}
	finalMergeMap := map[string]any{
		"swagger":  "2.0",
		"basePath": "/api/v1",
		"info": map[string]any{
			"title":       "Combined Gateway API Ecosystem",
			"version":     "1.0",
			"description": "Aggregated documentation for all microservices.",
		},
		"tags": []map[string]any{
			{"name": "auth", "description": "User registration, authentication, and session management"},
			{"name": "user", "description": "Profile management, account data retrieval, and user deletion logs"},
		},
		"paths":       make(map[string]any),
		"definitions": make(map[string]any),
	}
	services := []string{AuthService, BudgetService}
	for _, serviceName := range services {
		value, exist := s.DocsMap.Load(serviceName)
		if !exist {
			continue
		}
		dataDocs, ok := value.([]byte)
		if !ok {
			s.Logger.Error("failed to assert type key")
			continue
		}
		rawJson := string(dataDocs)
		title := cases.Title(language.AmericanEnglish)
		keyTitle := title.String(serviceName)
		rawJson = strings.ReplaceAll(rawJson, `"response.Response"`, `"response.`+keyTitle+`Response"`)
		rawJson = strings.ReplaceAll(rawJson, `"response.NegativeResponse"`, `"response.`+keyTitle+`NegativeResponse"`)
		rawJson = strings.ReplaceAll(rawJson, `#/definitions/response.Response`, `#/definitions/response.`+keyTitle+`Response`)
		rawJson = strings.ReplaceAll(rawJson, `#/definitions/response.NegativeResponse`, `#/definitions/response.`+keyTitle+`NegativeResponse`)
		var currentMap map[string]any
		if errUnmarshal := json.Unmarshal([]byte(rawJson), &currentMap); errUnmarshal != nil {
			s.Logger.Error("failed to unmarshal docs: " + errUnmarshal.Error())
			continue
		}
		if nextPaths, okNextPath := currentMap["paths"].(map[string]any); okNextPath {
			if basePaths, okBasePath := finalMergeMap["paths"].(map[string]any); okBasePath {
				for pathKey, pathValue := range nextPaths {
					basePaths[pathKey] = pathValue
				}
			}
		}
		if nextDefs, okNextDefs := currentMap["definitions"].(map[string]any); okNextDefs {
			if baseDefs, okBaseDefs := finalMergeMap["definitions"].(map[string]any); okBaseDefs {
				for defsKey, defsValue := range nextDefs {
					baseDefs[defsKey] = defsValue
				}
			}
		}
	}
	if dataMergeDocs, errMarshal := json.Marshal(finalMergeMap); errMarshal != nil {
		return nil, ErrFailedMergeDocs
	} else {
		return dataMergeDocs, nil
	}
}

func (s *ServiceSwaggerDocs) UpdateDocs() {
	s.DocsMap.Clear()
	var wg sync.WaitGroup
	for _, list := range s.ServiceList {
		wg.Add(1)
		go func(ls ListDocs) {
			defer wg.Done()
			resp, errGetDocs := http.Get(ls.Url)
			if errGetDocs != nil {
				s.Logger.Error("failed to get docs: " + ls.Service + ": " + errGetDocs.Error())
				return
			}
			defer func() {
				if errClose := resp.Body.Close(); errClose != nil {
					s.Logger.Error("failed to close: " + ls.Service + ": " + errClose.Error())
				}
			}()
			dataDocs, errRead := io.ReadAll(resp.Body)
			if errRead != nil {
				s.Logger.Error("failed to read docs: " + ls.Service + ": " + errRead.Error())
				return
			}
			s.DocsMap.Store(list.Service, dataDocs)
		}(list)
	}
	wg.Wait()
}

func (s *ServiceSwaggerDocs) PlanUpdateDocs(ctxCancel context.Context) {
	ticker := time.NewTicker(time.Hour * 1)
	defer ticker.Stop()
	for {
		select {
		case <-ctxCancel.Done():
			s.Logger.Info("graceful shutdown PlanUpdateDocs")
			return
		case <-ticker.C:
			s.UpdateDocs()
			s.Logger.Info("scheduled docs update")
		}
	}
}
