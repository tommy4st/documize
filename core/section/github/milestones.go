// Copyright 2016 Documize Inc. <legal@documize.com>. All rights reserved.
//
// This software (Documize Community Edition) is licensed under
// GNU AGPL v3 http://www.gnu.org/licenses/agpl-3.0.en.html
//
// You can operate outside the AGPL restrictions by purchasing
// Documize Enterprise Edition and obtaining a commercial license
// by contacting <sales@documize.com>.
//
// https://documize.com

package github

import (
	"fmt"
	"sort"
	"time"

	"github.com/documize/community/core/log"

	gogithub "github.com/google/go-github/github"
)

type githubMilestone struct {
	Repo         string `json:"repo"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	IsOpen       bool   `json:"isopen"`
	OpenIssues   int    `json:"openIssues"`
	ClosedIssues int    `json:"closedIssues"`
	CompleteMsg  string `json:"completeMsg"`
	DueDate      string `json:"dueDate"`
	UpdatedAt    string `json:"updatedAt"`
}

// sort milestones in order that that should be presented - by date updated.

type milestonesToSort []githubMilestone

func (s milestonesToSort) Len() int      { return len(s) }
func (s milestonesToSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s milestonesToSort) Less(i, j int) bool {
	if !s[i].IsOpen && s[j].IsOpen {
		return true
	}
	if s[i].IsOpen && !s[j].IsOpen {
		return false
	}
	// TODO this seems a very slow approach
	iDate, iErr := time.Parse(milestonesTimeFormat, s[i].UpdatedAt)
	log.IfErr(iErr)
	jDate, jErr := time.Parse(milestonesTimeFormat, s[j].UpdatedAt)
	log.IfErr(jErr)
	return iDate.Before(jDate)

}

const (
	tagMilestonesData    = "milestonesData"
	milestonesTimeFormat = "January 2 2006"

	rawMSsvg  = `<path d="M8 2H6V0h2v2zm4 5H2c-.55 0-1-.45-1-1V4c0-.55.45-1 1-1h10l2 2-2 2zM8 4H6v2h2V4zM6 16h2V8H6v8z"></path>`
	openMSsvg = `							
<span title="Open milestone">
	<svg height="16" width="14" version="1.1" viewBox="0 0 14 16">
		` + rawMSsvg + `
	</svg>
</span>
`
	closedMSsvg = `							
<span title="Closed milestone">
	<svg height="8" width="7" version="1.1" viewBox="0 0 14 16">
		` + rawMSsvg + `
	</svg>
</span>
`
)

func init() {
	reports[tagMilestonesData] = report{refreshMilestones, renderMilestones, `
<div class="section-github-render">
	<h3>Milestones</h3>
	<div class="github-board">
	<ul class="github-list">
		{{range $data := .Milestones}}
			<li class="github-commit-item">
				<a class="link" href="{{$data.URL}}">
					<div class="issue-avatar">
						{{if $data.IsOpen}}
							` + openMSsvg + `
						{{else}}
							` + closedMSsvg + `
						{{end}}
				  	</div>
					<div class="github-commit-body">
						<div class="github-commit-title"><span class="label-name">{{$data.Repo}} - {{$data.Name}}</span> </div>
						<div class="github-commit-meta">
						  {{$data.DueDate}} Last updated {{$data.UpdatedAt}}. 
						  {{$data.CompleteMsg}} complete {{$data.OpenIssues}} open {{$data.ClosedIssues}} closed
						</div>
					</div>
				</a>
				<div class="clearfix" />
			</li>
		{{end}}
	</ul>
	</div>
</div>
`}
}

func getMilestones(client *gogithub.Client, config *githubConfig) ([]githubMilestone, error) {

	ret := []githubMilestone{}

	hadRepo := make(map[string]bool)

	for _, orb := range config.Lists {
		if orb.Included {
			rName := orb.Owner + "/" + orb.Repo

			if !hadRepo[rName] {

				for _, state := range []string{"open", "closed"} {

					opts := &gogithub.MilestoneListOptions{
						Sort:        "updated",
						State:       state,
						ListOptions: gogithub.ListOptions{PerPage: config.BranchLines}}

					guff, _, err := client.Issues.ListMilestones(orb.Owner, orb.Repo, opts)

					if err != nil {
						return ret, err
					}

					for _, v := range guff {
						include := true
						if state == "closed" {
							if config.SincePtr != nil {
								if (*config.SincePtr).After(*v.ClosedAt) {
									include = false
								}
							}
						}
						if include {
							dd := "No due date."
							if v.DueOn != nil {
								// TODO refactor to add message in red if the milestone is overdue
								dd = "Due on " + (*v.DueOn).Format(milestonesTimeFormat) + "."
							}
							up := ""
							if v.UpdatedAt != nil {
								up = (*v.UpdatedAt).Format(milestonesTimeFormat)
							}

							ret = append(ret, githubMilestone{
								Repo:         rName,
								Name:         *v.Title,
								URL:          *v.HTMLURL,
								IsOpen:       *v.State == "open",
								OpenIssues:   *v.OpenIssues,
								ClosedIssues: *v.ClosedIssues,
								CompleteMsg:  fmt.Sprintf("%2.0f%%", float64(*v.ClosedIssues*100)/float64(*v.OpenIssues+*v.ClosedIssues)),
								DueDate:      dd,
								UpdatedAt:    up,
							})
						}
					}

				}
			}
			hadRepo[rName] = true
		}

	}

	sort.Stable(milestonesToSort(ret))

	return ret, nil

}

func refreshMilestones(gr *githubRender, config *githubConfig, client *gogithub.Client) (err error) {

	gr.Milestones, err = getMilestones(client, config)
	if err != nil {
		log.Error("unable to get github milestones", err)
		return err
	}
	gr.OpenMS = 0
	gr.ClosedMS = 0
	for _, v := range gr.Milestones {
		if v.IsOpen {
			gr.OpenMS++
		} else {
			gr.ClosedMS++
		}
	}

	return nil
}

func renderMilestones(payload *githubRender, c *githubConfig) error {
	return nil
}
