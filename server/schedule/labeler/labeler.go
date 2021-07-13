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

package labeler

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pingcap/log"
	"github.com/tikv/pd/pkg/errs"
	"github.com/tikv/pd/server/core"
	"go.uber.org/zap"
)

// RegionLabeler is utility to label regions.
type RegionLabeler struct {
	storage *core.Storage
	sync.RWMutex
	labelRules map[string]*LabelRule
}

// NewRegionLabeler creates a Labeler instance.
func NewRegionLabeler(storage *core.Storage) (*RegionLabeler, error) {
	l := &RegionLabeler{
		storage:    storage,
		labelRules: map[string]*LabelRule{},
	}

	if err := l.loadRules(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *RegionLabeler) loadRules() error {
	var toDelete []string
	err := l.storage.LoadRegionRules(func(k, v string) {
		var r LabelRule
		if err := json.Unmarshal([]byte(v), &r); err != nil {
			log.Error("failed to unmarshal label rule value", zap.String("rule-key", k), zap.String("rule-value", v), errs.ZapError(errs.ErrLoadRule))
			toDelete = append(toDelete, k)
			return
		}
		if err := l.adjustRule(&r); err != nil {
			log.Error("failed to adjust label rule", zap.String("rule-key", k), zap.String("rule-value", v), zap.Error(err))
			toDelete = append(toDelete, k)
			return
		}
		l.labelRules[r.ID] = &r
	})
	if err != nil {
		return err
	}
	for _, d := range toDelete {
		if err = l.storage.DeleteRegionRule(d); err != nil {
			return err
		}
	}
	return nil
}

func (l *RegionLabeler) adjustRule(rule *LabelRule) error {
	switch rule.RuleType {
	case KeyRange:
		data, ok := rule.Rule.(map[string]string)
		if !ok {
			return errs.ErrRegionRuleContent.FastGenByArgs("invalid rule type")
		}
		var r KeyRangeRule
		r.StartKeyHex, r.EndKeyHex = data["start_key"], data["end_key"]
		var err error
		r.StartKey, err = hex.DecodeString(r.StartKeyHex)
		if err != nil {
			return errs.ErrHexDecodingString.FastGenByArgs(r.StartKeyHex)
		}
		r.EndKey, err = hex.DecodeString(r.EndKeyHex)
		if err != nil {
			return errs.ErrHexDecodingString.FastGenByArgs(r.EndKeyHex)
		}
		if len(r.EndKey) > 0 && bytes.Compare(r.EndKey, r.StartKey) <= 0 {
			return errs.ErrRegionRuleContent.FastGenByArgs("endKey should be greater than startKey")
		}
		rule.Rule = r
	}
	return errs.ErrRegionRuleContent.FastGenByArgs(fmt.Sprintf("invalid rule type: %s", rule.RuleType))
}

// GetAllLabelRules returns all the rules.
func (l *RegionLabeler) GetAllLabelRules() []*LabelRule {
	l.RLock()
	defer l.RUnlock()
	rules := make([]*LabelRule, 0, len(l.labelRules))
	for _, rule := range l.labelRules {
		rules = append(rules, rule)
	}
	return rules
}

// GetLabelRule returns the Rule with the same ID.
func (l *RegionLabeler) GetLabelRule(id string) *LabelRule {
	l.RLock()
	defer l.RUnlock()
	return l.labelRules[id]
}

// SetLabelRule inserts or updates a LabelRule.
func (l *RegionLabeler) SetLabelRule(rule *LabelRule) error {
	l.Lock()
	defer l.Unlock()
	if err := l.adjustRule(rule); err != nil {
		return err
	}
	if err := l.storage.SaveRegionRule(rule.ID, rule); err != nil {
		return err
	}
	l.labelRules[rule.ID] = rule
	return nil
}

// DeleteRule removes a LabelRule.
func (l *RegionLabeler) DeleteLabelRule(id string) error {
	l.Lock()
	defer l.Unlock()
	if err := l.storage.DeleteRegionRule(id); err != nil {
		return err
	}
	delete(l.labelRules, id)
	return nil
}

// GetRegionLabel returns the label of the region for a key.
func (l *RegionLabeler) GetRegionLabel(region *core.RegionInfo, key string) string {
	l.RLock()
	defer l.RUnlock()
	for _, rule := range l.labelRules {
		if rule.IsMatch(region) {
			for _, label := range rule.Labels {
				if label.Key == key {
					return label.Value
				}
			}
		}
	}
	return ""
}

// GetRegionLabelsreturns the labels of the region.
func (l *RegionLabeler) GetRegionLabels(region *core.RegionInfo) []*RegionLabel {
	l.RLock()
	defer l.RUnlock()
	var result []*RegionLabel
	for _, rule := range l.labelRules {
		if rule.IsMatch(region) {
			for _, label := range rule.Labels {
				result = append(result, &RegionLabel{
					Key:   label.Key,
					Value: label.Value,
				})
			}
		}
	}
	return result
}