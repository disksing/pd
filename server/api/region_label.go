// Copyright 2021 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tikv/pd/pkg/apiutil"
	"github.com/tikv/pd/pkg/errs"
	"github.com/tikv/pd/server"
	"github.com/tikv/pd/server/schedule/labeler"
	"github.com/unrolled/render"
)

type regionLabelHandler struct {
	svr *server.Server
	rd  *render.Render
}

func newRegionLabelHandler(s *server.Server, rd *render.Render) *regionLabelHandler {
	return &regionLabelHandler{
		svr: s,
		rd:  rd,
	}
}

// @Tags region_label
// @Summary List all label rules of cluster.
// @Produce json
// @Success 200 {array} labeler.LabelRule
// @Router /config/region-label/rule [get]
func (h *regionLabelHandler) GetAllRules(w http.ResponseWriter, r *http.Request) {
	cluster := getCluster(r)
	rules := cluster.GetRegionLabeler().GetAllLabelRules()
	h.rd.JSON(w, http.StatusOK, rules)
}

// @Tags region_label
// @Summary Get label rule of cluster by id.
// @Param id path string true "Rule Id"
// @Produce json
// @Success 200 {object} labeler.LabelRule
// @Failure 404 {string} string "The rule does not exist."
// @Router /config/region-label/rule/{id} [get]
func (h *regionLabelHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	cluster := getCluster(r)
	id := mux.Vars(r)["id"]
	rule := cluster.GetRegionLabeler().GetLabelRule(id)
	if rule == nil {
		h.rd.JSON(w, http.StatusNotFound, nil)
		return
	}
	h.rd.JSON(w, http.StatusOK, rule)
}

// @Tags region_label
// @Summary Update region label rule of cluster.
// @Accept json
// @Param rule body labeler.LabelRule true "Parameters of label rule"
// @Produce json
// @Success 200 {string} string "Update rule successfully."
// @Failure 400 {string} string "The input is invalid."
// @Failure 500 {string} string "PD server failed to proceed the request."
// @Router /config/region-label/rule [post]
func (h *regionLabelHandler) SetRule(w http.ResponseWriter, r *http.Request) {
	cluster := getCluster(r)
	var rule labeler.LabelRule
	if err := apiutil.ReadJSONRespondError(h.rd, w, r.Body, &rule); err != nil {
		return
	}
	if err := cluster.GetRegionLabeler().SetLabelRule(&rule); err != nil {
		if errs.ErrRegionRuleContent.Equal(err) || errs.ErrHexDecodingString.Equal(err) {
			h.rd.JSON(w, http.StatusBadRequest, err.Error())
		} else {
			h.rd.JSON(w, http.StatusInternalServerError, err.Error())
		}
	}
	h.rd.JSON(w, http.StatusOK, "Update region label rule successfully.")
}

// @Tags region_label
// @Summary Get label of a region.
// @Param id path integer true "Region Id"
// @Param key path string true "Label key"
// @Produce json
// @Success 200 {string} string
// @Failure 400 {string} string "The input is invalid."
// @Failure 404 {string} string "The region does not exist."
// @Router /region/id/{id}/label/{key} [get]
func (h *regionLabelHandler) GetRegionLabel(w http.ResponseWriter, r *http.Request) {
	cluster := getCluster(r)
	regionID, labelKey := mux.Vars(r)["id"], mux.Vars(r)["key"]
	id, err := strconv.ParseUint(regionID, 10, 64)
	if err != nil {
		h.rd.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	region := cluster.GetRegion(id)
	if region == nil {
		h.rd.JSON(w, http.StatusNotFound, nil)
		return
	}
	labelValue := cluster.GetRegionLabeler().GetRegionLabel(region, labelKey)
	h.rd.JSON(w, http.StatusOK, labelValue)
}

// @Tags region_label
// @Summary Get labels of a region.
// @Param id path integer true "Region Id"
// @Produce json
// @Success 200 {string} string
// @Failure 400 {string} string "The input is invalid."
// @Failure 404 {string} string "The region does not exist."
// @Router /region/id/{id}/labels [get]
func (h *regionLabelHandler) GetRegionLabels(w http.ResponseWriter, r *http.Request) {
	cluster := getCluster(r)
	regionID, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		h.rd.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	region := cluster.GetRegion(regionID)
	if region == nil {
		h.rd.JSON(w, http.StatusNotFound, nil)
		return
	}
	labels := cluster.GetRegionLabeler().GetRegionLabels(region)
	h.rd.JSON(w, http.StatusOK, labels)
}