package client

import (
	"fmt"
	"time"

	"github.com/adevinta/go-log-toolkit"
	"github.com/cenk/backoff"
	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
)

func p[T any](v T) *T {
	return &v
}

// DashboardClient defines operations for uploading and updating dashboards
// in a Grafana instance.
type DashboardClient interface {
	// UploadDashboard creates or updates a dashboard in Grafana.
	UploadDashboard(dashboard *Dashboard) error

	// GetDashboard retrieves a dashboard by its UID.
	GetDashboard(uid string) (*Dashboard, error)

	// DeleteDashboard removes a dashboard identified by its UID.
	DeleteDashboard(uid string) error

	// EnsureFolder creates a folder if it doesn't exist or returns existing folder.
	EnsureFolder(rootFolder *Folder, folder string) (*Folder, error)

	// GetDataSource retrieves a datasource by its name.
	GetDataSource(name string) (*Datasource, error)

	// ListDashboardIDsInFolder lists all dashboards in a folder.
	ListDashboardIDsInFolder(folderUID string) ([]string, error)
}

type JSON interface{}

// Folder represents a Grafana folder with its UID and title
type Folder struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
}

// Dashboard represents a Grafana dashboard with its metadata and content
type Dashboard struct {
	UID       string `json:"uid"`
	Title     string `json:"title"`
	FolderUID string
	Dashboard JSON
	Meta      *models.DashboardMeta
}

type Datasource = models.DataSource

func (sc *StackClient) GetDataSource(name string) (*Datasource, error) {

	res, err := sc.httpApi.Datasources.GetDataSourceByName(name)

	if err != nil {
		return nil, fmt.Errorf("failed to get datasource for %s: %w", name, err)
	}

	if res.Payload == nil {
		return nil, fmt.Errorf("received no datasource data for: %s", name)
	}

	return res.Payload, nil
}

func (sc *StackClient) DeleteDashboard(uid string) error {

	_, err := sc.httpApi.Dashboards.DeleteDashboardByUID(uid)

	if err != nil {
		return fmt.Errorf("failed to delete dashboard %s: %w", uid, err)
	}

	return nil
}

func (sc *StackClient) UploadDashboard(dashboard *Dashboard) error {

	saveDashboardCmd := &models.SaveDashboardCommand{
		Dashboard: dashboard.Dashboard,
		FolderUID: dashboard.FolderUID,
		Overwrite: true,
		IsFolder:  false,
		Message:   "toolkit/grafana automated dashboard upload",
	}

	_, err := sc.httpApi.Dashboards.PostDashboard(saveDashboardCmd)
	if err != nil {
		return fmt.Errorf("failed to updload dashboard %s: %w", dashboard.UID, err)
	}

	return nil
}

func (sc *StackClient) GetDashboard(uid string) (*Dashboard, error) {

	res, err := sc.httpApi.Dashboards.GetDashboardByUID(uid)

	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard %s: %w", uid, err)
	}

	if res.Payload == nil || res.Payload.Dashboard == nil {
		return nil, fmt.Errorf("received no dashboard data for uid: %s", uid)
	}

	return &Dashboard{
		UID:       uid,
		Dashboard: res.Payload.Dashboard,
		Meta:      res.Payload.Meta,
	}, nil
}

func (sc *StackClient) ListDashboardIDsInFolder(folderUID string) ([]string, error) {
	params := search.NewSearchParams().
		WithFolderUIDs([]string{folderUID}).
		WithType(p("dash-db"))

	// TODO: handle pagination.
	// Inspecting the Search results there is no easy way to retrieve the
	// pagination options.
	// This means it is likely that folder with many dashboards will not be
	// fully listed.
	res, err := sc.httpApi.Search.Search(params)

	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards in folder %s: %w", folderUID, err)
	}

	dashboardUIDs := make([]string, 0, len(res.Payload))

	for _, hit := range res.Payload {
		dashboardUIDs = append(dashboardUIDs, hit.UID)
	}

	return dashboardUIDs, nil
}

func (sc *StackClient) GetFolder(rootFolder *Folder, folderName string) (*Folder, error) {

	params := folders.NewGetFoldersParams()
	if rootFolder != nil {
		params.ParentUID = &rootFolder.UID
	}
	foldersRes, err := sc.httpApi.Folders.GetFolders(params)

	if err != nil {
		return nil, fmt.Errorf("failed to get folders for  %s: %w", folderName, err)
	}

	log.DefaultLogger.WithField("folders", len(foldersRes.Payload)).Debugf("done listing folders")

	for _, f := range foldersRes.Payload {
		log.DefaultLogger.WithField("folder", f.Title).WithField("searched", folderName).Tracef("matching folder")
		if f.Title == folderName {
			return &Folder{
				UID:   f.UID,
				Title: f.Title,
			}, nil
		}
	}

	log.DefaultLogger.WithField("searched", folderName).Debugf("not found")

	return nil, nil
}

func (sc *StackClient) EnsureFolder(rootFolder *Folder, folderName string) (*Folder, error) {

	folder, err := sc.GetFolder(rootFolder, folderName)

	if err != nil {
		return nil, fmt.Errorf("failed to get folders for %s: %w", folderName, err)
	}

	log.DefaultLogger.WithField("folder", folder). WithField("searched", folderName). Tracef("found folder")

	if folder != nil {
		return folder, nil
	}

	log.DefaultLogger.WithField("folder", folderName).Debugf("creating new folder")

	createFolderCmd := &models.CreateFolderCommand{Title: folderName}
	if rootFolder != nil {
		createFolderCmd.ParentUID = rootFolder.UID
	}
	createRes, err := sc.httpApi.Folders.CreateFolder(createFolderCmd)

	if err != nil {
		return nil, fmt.Errorf("failed to create folder %s: %w", folderName, err)
	}

	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = time.Minute
	retry.MaxInterval = 10 * time.Second

	err = backoff.Retry(func() error {
		folder, err := sc.GetFolder(rootFolder, folderName)
		if err != nil {
			log.DefaultLogger.WithError(err).WithField("folder", folderName).Debugf("failed to get folder")
			return err
		}
		if folder != nil {
			return nil
		}

		return fmt.Errorf("folder not found")
	}, retry)

	if err != nil {
		return nil, fmt.Errorf("failed to create folder %s: %w", folderName, err)
	}

	return &Folder{
		UID:   createRes.Payload.UID,
		Title: createRes.Payload.Title,
	}, nil
}
