package model

import "github.com/QuantumNous/new-api/common"

type ContributorModelUsage struct {
	ModelName    string `json:"model_name"`
	RequestCount int64  `json:"request_count"`
	TokenUsed    int64  `json:"token_used"`
}

type ContributorOverview struct {
	ChannelCount        int64                   `json:"channel_count"`
	EnabledChannelCount int64                   `json:"enabled_channel_count"`
	RequestCount        int64                   `json:"request_count"`
	TokenUsed           int64                   `json:"token_used"`
	ModelRanking        []ContributorModelUsage `json:"model_ranking"`
}

func GetContributorOverview(contributorID int) (*ContributorOverview, error) {
	overview := &ContributorOverview{ModelRanking: []ContributorModelUsage{}}
	if err := contributorScopedChannels(DB, contributorID).Count(&overview.ChannelCount).Error; err != nil {
		return nil, err
	}
	if err := contributorScopedChannels(DB, contributorID).
		Where("status = ?", common.ChannelStatusEnabled).
		Count(&overview.EnabledChannelCount).Error; err != nil {
		return nil, err
	}

	var channelIDs []int
	if err := contributorScopedChannels(DB, contributorID).Pluck("id", &channelIDs).Error; err != nil {
		return nil, err
	}
	if len(channelIDs) == 0 {
		return overview, nil
	}

	var ranking []ContributorModelUsage
	query := DB.Model(&QuotaData{}).
		Select("model_name, COALESCE(SUM(count), 0) AS request_count, COALESCE(SUM(token_used), 0) AS token_used").
		Where("channel_id IN ?", channelIDs).
		Group("model_name").
		Order("request_count DESC")
	if err := query.Scan(&ranking).Error; err != nil {
		return nil, err
	}
	for _, item := range ranking {
		overview.RequestCount += item.RequestCount
		overview.TokenUsed += item.TokenUsed
	}
	overview.ModelRanking = ranking
	return overview, nil
}
