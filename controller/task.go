package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func GetAllTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 解析其他查询参数
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ChannelID:      c.Query("channel_id"),
	}

	items := model.TaskGetAllTasks(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllTasks(queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(adminTasksToDto(items))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	items := model.TaskGetAllUserTask(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTask(userId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(userTasksToDto(c, items))
	common.ApiSuccess(c, pageInfo)
}

func adminTasksToDto(tasks []*model.Task) []*dto.TaskDto {
	userIdMap := make(map[int]*model.UserBase)
	userIds := types.NewSet[int]()
	for _, task := range tasks {
		userIds.Add(task.UserId)
	}
	for _, userId := range userIds.Items() {
		cacheUser, err := model.GetUserCache(userId)
		if err == nil {
			userIdMap[userId] = cacheUser
		}
	}
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		result[i] = relay.TaskModel2Dto(task)
		if user, ok := userIdMap[task.UserId]; ok {
			result[i].Username = user.Username
		}
	}
	return result
}

func userTasksToDto(c *gin.Context, tasks []*model.Task) []*dto.UserTaskDto {
	result := make([]*dto.UserTaskDto, len(tasks))
	for i, task := range tasks {
		result[i] = relay.TaskModel2UserDto(c, task)
	}
	return result
}
