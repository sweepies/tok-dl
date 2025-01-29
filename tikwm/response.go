package tikwm

type ApiResponse struct {
	Code          int     `json:"code,omitempty"`
	Msg           string  `json:"msg"`
	ProcessedTime float64 `json:"processed_time,omitempty"`
	Data          struct {
		ID             string `json:"id,omitempty"`
		Region         string `json:"region,omitempty"`
		Title          string `json:"title,omitempty"`
		Cover          string `json:"cover,omitempty"`
		AiDynamicCover string `json:"ai_dynamic_cover,omitempty"`
		OriginCover    string `json:"origin_cover,omitempty"`
		Duration       int    `json:"duration,omitempty"`
		Play           string `json:"play,omitempty"`
		Hdplay         string `json:"hdplay,omitempty"`
		Wmplay         string `json:"wmplay,omitempty"`
		Size           int    `json:"size,omitempty"`
		WmSize         int    `json:"wm_size,omitempty"`
		HdSize         int    `json:"hd_size,omitempty"`
		Music          string `json:"music,omitempty"`
		MusicInfo      struct {
			ID       string `json:"id,omitempty"`
			Title    string `json:"title,omitempty"`
			Play     string `json:"play,omitempty"`
			Cover    string `json:"cover,omitempty"`
			Author   string `json:"author,omitempty"`
			Original bool   `json:"original,omitempty"`
			Duration int    `json:"duration,omitempty"`
			Album    string `json:"album,omitempty"`
		} `json:"music_info,omitempty"`
		PlayCount     int `json:"play_count,omitempty"`
		DiggCount     int `json:"digg_count,omitempty"`
		CommentCount  int `json:"comment_count,omitempty"`
		ShareCount    int `json:"share_count,omitempty"`
		DownloadCount int `json:"download_count,omitempty"`
		CollectCount  int `json:"collect_count,omitempty"`
		CreateTime    int `json:"create_time,omitempty"`
		Anchors       []struct {
			Actions      []any  `json:"actions,omitempty"`
			AnchorStrong any    `json:"anchor_strong,omitempty"`
			ComponentKey string `json:"component_key,omitempty"`
			Description  string `json:"description,omitempty"`
			Extra        string `json:"extra,omitempty"`
			GeneralType  int    `json:"general_type,omitempty"`
			Icon         struct {
				Height    int      `json:"height,omitempty"`
				URI       string   `json:"uri,omitempty"`
				URLList   []string `json:"url_list,omitempty"`
				URLPrefix any      `json:"url_prefix,omitempty"`
				Width     int      `json:"width,omitempty"`
			} `json:"icon,omitempty"`
			ID        string `json:"id,omitempty"`
			Keyword   string `json:"keyword,omitempty"`
			LogExtra  string `json:"log_extra,omitempty"`
			Schema    string `json:"schema,omitempty"`
			Thumbnail struct {
				Height    int      `json:"height,omitempty"`
				URI       string   `json:"uri,omitempty"`
				URLList   []string `json:"url_list,omitempty"`
				URLPrefix any      `json:"url_prefix,omitempty"`
				Width     int      `json:"width,omitempty"`
			} `json:"thumbnail,omitempty"`
			Type int `json:"type,omitempty"`
		} `json:"anchors,omitempty"`
		AnchorsExtras string `json:"anchors_extras,omitempty"`
		IsAd          bool   `json:"is_ad,omitempty"`
		CommerceInfo  struct {
			AdvPromotable          bool `json:"adv_promotable,omitempty"`
			AuctionAdInvited       bool `json:"auction_ad_invited,omitempty"`
			BrandedContentType     int  `json:"branded_content_type,omitempty"`
			WithCommentFilterWords bool `json:"with_comment_filter_words,omitempty"`
		} `json:"commerce_info,omitempty"`
		CommercialVideoInfo string `json:"commercial_video_info,omitempty"`
		ItemCommentSettings int    `json:"item_comment_settings,omitempty"`
		MentionedUsers      string `json:"mentioned_users,omitempty"`
		Author              struct {
			ID       string `json:"id,omitempty"`
			UniqueID string `json:"unique_id,omitempty"`
			Nickname string `json:"nickname,omitempty"`
			Avatar   string `json:"avatar,omitempty"`
		} `json:"author,omitempty"`
		Images []string `json:"images,omitempty"`
	} `json:"data,omitempty"`
}
