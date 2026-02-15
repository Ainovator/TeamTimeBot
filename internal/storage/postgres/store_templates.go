package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) UpsertPollTemplate(
	ctx context.Context,
	chatID int64,
	name, question string,
	options []string,
) (*PollTemplate, error) {
	defaultCounted := []int{}
	if len(options) > 0 {
		defaultCounted = []int{0}
	}
	return s.UpsertPollTemplateWithCounted(ctx, chatID, name, question, options, defaultCounted)
}

func (s *Store) UpsertPollTemplateWithCounted(
	ctx context.Context,
	chatID int64,
	name, question string,
	options []string,
	countedOptions []int,
) (*PollTemplate, error) {
	weights := make([]int, 0, len(options))
	for range options {
		weights = append(weights, 1)
	}
	return s.UpsertPollTemplateWithCountedAndWeights(ctx, chatID, name, question, options, countedOptions, weights)
}

func normalizeOptionWeights(optionsLen int, weights []int) []int {
	if optionsLen <= 0 {
		return []int{}
	}
	out := make([]int, optionsLen)
	for i := 0; i < optionsLen; i++ {
		w := 1
		if i < len(weights) {
			w = weights[i]
		}
		if w <= 0 {
			w = 1
		}
		out[i] = w
	}
	return out
}

func (s *Store) UpsertPollTemplateWithCountedAndWeights(
	ctx context.Context,
	chatID int64,
	name, question string,
	options []string,
	countedOptions []int,
	optionWeights []int,
) (*PollTemplate, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	sanitizedOptions := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			sanitizedOptions = append(sanitizedOptions, option)
		}
	}
	if len(sanitizedOptions) < 2 {
		return nil, errors.New("at least 2 options are required")
	}

	name = strings.TrimSpace(name)
	question = strings.TrimSpace(question)
	if name == "" {
		return nil, errors.New("template name is required")
	}
	if question == "" {
		return nil, errors.New("question is required")
	}

	payload, err := json.Marshal(sanitizedOptions)
	if err != nil {
		return nil, err
	}

	normalizedCounted := normalizeCountedOptionIndexes(len(sanitizedOptions), countedOptions)
	countedPayload, err := json.Marshal(normalizedCounted)
	if err != nil {
		return nil, err
	}

	normalizedWeights := normalizeOptionWeights(len(sanitizedOptions), optionWeights)
	weightsPayload, err := json.Marshal(normalizedWeights)
	if err != nil {
		return nil, err
	}

	template := PollTemplate{
		GroupID:        group.ID,
		Name:           name,
		Question:       question,
		Options:        datatypes.JSON(payload),
		CountedOptions: datatypes.JSON(countedPayload),
		OptionWeights:  datatypes.JSON(weightsPayload),
		IsActive:       true,
	}

	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "group_id"},
				{Name: "name"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"question":        question,
				"options":         datatypes.JSON(payload),
				"counted_options": datatypes.JSON(countedPayload),
				"option_weights":  datatypes.JSON(weightsPayload),
				"is_active":       true,
				"updated_at":      gorm.Expr("NOW()"),
			}),
		}).
		Create(&template).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ?", group.ID, name).
		First(&template).Error; err != nil {
		return nil, err
	}

	return &template, nil
}

func (s *Store) EnsureDefaultRegistrationTemplate(ctx context.Context, chatID int64) (*TemplateDetails, error) {
	const name = "Регистрация"
	const question = "Регистрация: выбери вариант"
	options := []string{"Зарегистрироваться", "Не буду"}
	counted := []int{0}

	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	// Do not overwrite an existing template if the admin edited it.
	var existing PollTemplate
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, name).
		First(&existing).Error; err == nil {
		return s.GetTemplateByName(ctx, chatID, name)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.UpsertPollTemplateWithCounted(ctx, chatID, name, question, options, counted); err != nil {
		return nil, err
	}
	return s.GetTemplateByName(ctx, chatID, name)
}

func isSystemTemplateName(templateName string) bool {
	// Keep this simple and explicit for now.
	// If we later add more system templates, they can be listed here.
	return strings.EqualFold(strings.TrimSpace(templateName), "Регистрация")
}

func (s *Store) DeleteTemplateByName(ctx context.Context, chatID int64, templateName string) error {
	if isSystemTemplateName(templateName) {
		return errors.New("нельзя удалить системный шаблон \"Регистрация\"")
	}

	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var template PollTemplate
		if err := tx.
			Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, templateName).
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("template not found")
			}
			return err
		}

		if err := tx.Model(&PollTemplate{}).
			Where("id = ?", template.ID).
			Updates(map[string]interface{}{
				"is_active":  false,
				"updated_at": gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&PollSchedule{}).
			Where("template_id = ?", template.ID).
			Updates(map[string]interface{}{
				"is_active":  false,
				"updated_at": gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *Store) GetGroupSnapshot(ctx context.Context, chatID int64) ([]TemplateView, []ScheduleView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, nil, err
	}

	var templates []TemplateView
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name, question, COALESCE(jsonb_array_length(counted_options), 0) AS counted_options_count").
		Where("group_id = ? AND is_active = TRUE", group.ID).
		Order("name ASC").
		Scan(&templates).Error; err != nil {
		return nil, nil, err
	}

	var schedules []ScheduleView
	if err := s.db.WithContext(ctx).
		Table("poll_schedules ps").
		Select("t.name AS template_name, ps.cron_expr AS send_at").
		Joins("JOIN poll_templates t ON t.id = ps.template_id").
		Where("ps.group_id = ? AND ps.is_active = TRUE AND t.is_active = TRUE", group.ID).
		Order("t.name ASC").
		Scan(&schedules).Error; err != nil {
		return nil, nil, err
	}

	for i := range schedules {
		schedules[i].SendAt = FormatScheduleExprForDisplay(schedules[i].SendAt)
	}

	return templates, schedules, nil
}

func (s *Store) ListTemplateNamesByChatID(ctx context.Context, chatID int64) ([]string, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type row struct {
		Name string
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name").
		Where("group_id = ? AND is_active = TRUE", group.ID).
		Order("name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]string, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.Name)
	}
	return result, nil
}

func (s *Store) GetTemplateByName(ctx context.Context, chatID int64, name string) (*TemplateDetails, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var row struct {
		Name           string
		Question       string
		Options        datatypes.JSON
		CountedOptions datatypes.JSON
		OptionWeights  datatypes.JSON
	}
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name, question, options, counted_options, COALESCE(option_weights, '[]'::jsonb) AS option_weights").
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, name).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}

	var options []string
	if err := json.Unmarshal(row.Options, &options); err != nil {
		return nil, err
	}
	var countedOptions []int
	if err := json.Unmarshal(row.CountedOptions, &countedOptions); err != nil {
		return nil, err
	}
	var optionWeights []int
	_ = json.Unmarshal(row.OptionWeights, &optionWeights)

	return &TemplateDetails{
		Name:           row.Name,
		Question:       row.Question,
		Options:        options,
		CountedOptions: normalizeCountedOptionIndexes(len(options), countedOptions),
		OptionWeights:  normalizeOptionWeights(len(options), optionWeights),
	}, nil
}

func (s *Store) UpdateTemplateByName(
	ctx context.Context,
	chatID int64,
	currentName, newName, question string,
	options []string,
) (*TemplateDetails, error) {
	defaultCounted := []int{}
	if len(options) > 0 {
		defaultCounted = []int{0}
	}
	return s.UpdateTemplateByNameWithCounted(ctx, chatID, currentName, newName, question, options, defaultCounted)
}

func (s *Store) UpdateTemplateByNameWithCounted(
	ctx context.Context,
	chatID int64,
	currentName, newName, question string,
	options []string,
	countedOptions []int,
) (*TemplateDetails, error) {
	weights := make([]int, 0, len(options))
	for range options {
		weights = append(weights, 1)
	}
	return s.UpdateTemplateByNameWithCountedAndWeights(ctx, chatID, currentName, newName, question, options, countedOptions, weights)
}

func (s *Store) UpdateTemplateByNameWithCountedAndWeights(
	ctx context.Context,
	chatID int64,
	currentName, newName, question string,
	options []string,
	countedOptions []int,
	optionWeights []int,
) (*TemplateDetails, error) {
	// Prevent renaming of system templates to avoid breaking integrations
	// that rely on a stable name.
	if isSystemTemplateName(currentName) && strings.TrimSpace(newName) != "" && !strings.EqualFold(strings.TrimSpace(newName), strings.TrimSpace(currentName)) {
		return nil, errors.New("нельзя переименовать системный шаблон \"Регистрация\"")
	}

	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	currentName = strings.TrimSpace(currentName)
	newName = strings.TrimSpace(newName)
	question = strings.TrimSpace(question)
	if currentName == "" {
		return nil, errors.New("current template name is required")
	}
	if newName == "" {
		newName = currentName
	}
	if question == "" {
		return nil, errors.New("question is required")
	}

	sanitizedOptions := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			sanitizedOptions = append(sanitizedOptions, option)
		}
	}
	if len(sanitizedOptions) < 2 {
		return nil, errors.New("at least 2 options are required")
	}

	payload, err := json.Marshal(sanitizedOptions)
	if err != nil {
		return nil, err
	}
	normalizedCounted := normalizeCountedOptionIndexes(len(sanitizedOptions), countedOptions)
	countedPayload, err := json.Marshal(normalizedCounted)
	if err != nil {
		return nil, err
	}

	normalizedWeights := normalizeOptionWeights(len(sanitizedOptions), optionWeights)
	weightsPayload, err := json.Marshal(normalizedWeights)
	if err != nil {
		return nil, err
	}

	var result TemplateDetails
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var template PollTemplate
		if err := tx.
			Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, currentName).
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("template not found")
			}
			return err
		}

		var conflicts int64
		if err := tx.Model(&PollTemplate{}).
			Where("group_id = ? AND name = ? AND is_active = TRUE AND id <> ?", group.ID, newName, template.ID).
			Count(&conflicts).Error; err != nil {
			return err
		}
		if conflicts > 0 {
			return errors.New("template name already exists")
		}

		if err := tx.Model(&PollTemplate{}).
			Where("id = ?", template.ID).
			Updates(map[string]interface{}{
				"name":            newName,
				"question":        question,
				"options":         datatypes.JSON(payload),
				"counted_options": datatypes.JSON(countedPayload),
				"option_weights":  datatypes.JSON(weightsPayload),
				"is_active":       true,
				"updated_at":      gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		var row struct {
			Name           string
			Question       string
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
			OptionWeights  datatypes.JSON
		}
		if err := tx.
			Table("poll_templates").
			Select("name, question, options, counted_options, COALESCE(option_weights, '[]'::jsonb) AS option_weights").
			Where("id = ?", template.ID).
			First(&row).Error; err != nil {
			return err
		}

		var parsedOptions []string
		if err := json.Unmarshal(row.Options, &parsedOptions); err != nil {
			return err
		}
		var parsedCounted []int
		if err := json.Unmarshal(row.CountedOptions, &parsedCounted); err != nil {
			return err
		}
		var parsedWeights []int
		_ = json.Unmarshal(row.OptionWeights, &parsedWeights)

		result = TemplateDetails{
			Name:           row.Name,
			Question:       row.Question,
			Options:        parsedOptions,
			CountedOptions: normalizeCountedOptionIndexes(len(parsedOptions), parsedCounted),
			OptionWeights:  normalizeOptionWeights(len(parsedOptions), parsedWeights),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
