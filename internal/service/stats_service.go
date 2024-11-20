package service

import (
	"crowdfunding-backend/internal/repository/interfaces"
)

type StatsService struct {
	userRepo    interfaces.UserRepository
	projectRepo interfaces.ProjectRepository
}

func NewStatsService(userRepo interfaces.UserRepository, projectRepo interfaces.ProjectRepository) *StatsService {
	return &StatsService{
		userRepo:    userRepo,
		projectRepo: projectRepo,
	}
}

func (s *StatsService) GetSystemStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	userCount, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	stats["total_users"] = userCount

	// 由于我们移除了 Count 方法，我们需要使用 ListProjects 来获取项目总数
	projects, err := s.projectRepo.ListProjects(1, 1000000) // 使用一个很大的数字来获取所有项目
	if err != nil {
		return nil, err
	}
	stats["total_projects"] = len(projects)

	// 计算成功的项目数量
	var successfulProjects int
	for _, project := range projects {
		if project.Status == "successful" {
			successfulProjects++
		}
	}
	stats["successful_projects"] = successfulProjects

	// 计算总支持金额
	var totalPledged float64
	for _, project := range projects {
		totalPledged += project.TotalAmount // 假设 Project 结构体有 TotalAmount 字段
	}
	stats["total_pledged_amount"] = totalPledged

	return stats, nil
}
