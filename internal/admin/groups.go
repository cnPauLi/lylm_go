package admin

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// groupPage 分组管理模板数据
type groupPage struct {
	Set      string
	Pwds     []models.Pwd
	Edit     *models.Group
	IconType string
	IconVal  string
}

// TableGroupData 分组列表片段数据
type TableGroupData struct {
	Total     int64
	Groups    []models.Group
	PwdMap    map[int]string
	LinkCount map[int]int
}

// GroupPage 分组管理 group
func (h *Handler) GroupPage(c *gin.Context) {
	set := c.Query("set")
	if c.Request.Method == http.MethodPost {
		switch set {
		case "add_submit":
			h.groupAdd(c)
			return
		case "edit_submit":
			h.groupEdit(c)
			return
		case "del":
			if err := h.GroupSvc.Delete(postInt(c, "group_id")); err != nil {
				h.Log.Errorf("%v", err)
			}
			c.Status(http.StatusOK)
			return
		case "on":
			h.setGroupStatus(c, 1)
			return
		case "off":
			h.setGroupStatus(c, 0)
			return
		case "sort":
			h.groupSort(c)
			return
		}
	}

	list, err := h.GroupSvc.List()
	if err != nil {
		h.Log.Errorf("查询分组失败: %v", err)
	}
	gp := &groupPage{Set: set}
	if list != nil {
		gp.Pwds = list.Pwds
	}
	if set == "edit" {
		row, err := h.GroupSvc.Get(queryInt(c, "id"))
		if err != nil {
			h.renderHTML(c, alert("提示", "当前记录不存在！", "./group"))
			return
		}
		gp.Edit = &row
		gp.IconType, gp.IconVal = groupIconType(row.GroupIcon)
	}
	data := h.newData(c, "分组管理", "group", "group")
	data.Page = gp
	h.render(c, "group.html", data)
}

// groupIconType 判断图标类型并提取图片地址（展示辅助，保留在 admin 包）
func groupIconType(icon string) (string, string) {
	if strings.Contains(strings.ToLower(icon), "<svg") {
		return "svg", icon
	}
	if m := imgSrcRe.FindStringSubmatch(icon); m != nil {
		return "image", m[1]
	}
	return "image", icon
}

var imgSrcRe = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

func (h *Handler) groupAdd(c *gin.Context) {
	rec, err := h.GroupSvc.Create(service.GroupInput{
		Name: c.PostForm("group_name"),
		Icon: c.PostForm("group_icon"),
		Pwd:  postInt(c, "group_pwd"),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			h.renderHTML(c, notify("保存错误,请确保带星号的都不为空！", "danger", 3000))
			return
		}
		h.Log.Errorf("新增分组失败: %v", err)
		h.renderHTML(c, notify("添加分组失败", "danger", 3000))
		return
	}
	h.renderHTML(c, alert("成功", "添加分组 "+htmlEscape(rec.GroupName)+" 成功！", "./group"))
}

func (h *Handler) groupEdit(c *gin.Context) {
	rec, err := h.GroupSvc.Update(queryInt(c, "id"), service.GroupInput{
		Name: c.PostForm("group_name"),
		Icon: c.PostForm("group_icon"),
		Pwd:  postInt(c, "group_pwd"),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			h.renderHTML(c, notify("当前记录不存在！", "danger", 3000))
			return
		case errors.Is(err, service.ErrInvalidInput):
			h.renderHTML(c, notify("保存错误,请确保带星号的都不为空！", "danger", 3000))
			return
		default:
			h.Log.Errorf("修改分组失败: %v", err)
			h.renderHTML(c, notify("修改分组失败", "danger", 3000))
			return
		}
	}
	h.renderHTML(c, alert("成功", "修改分组 "+htmlEscape(rec.GroupName)+" 成功！", "./group"))
}

func (h *Handler) setGroupStatus(c *gin.Context, status int) {
	if err := h.GroupSvc.SetStatus(postInt(c, "group_id"), status); err != nil {
		h.Log.Errorf("更新分组状态失败: %v", err)
	}
	c.Status(http.StatusOK)
}

func (h *Handler) groupSort(c *gin.Context) {
	ids := make([]int, 0)
	for _, s := range postStrings(c, "groups") {
		if id, err := atoi(s); err == nil {
			ids = append(ids, id)
		}
	}
	if err := h.GroupSvc.Sort(ids); err != nil {
		h.Log.Errorf("更新分组排序失败: %v", err)
	}
	c.Status(http.StatusOK)
}

// TableGroup table_group 片段
func (h *Handler) TableGroup(c *gin.Context) {
	list, err := h.GroupSvc.List()
	if err != nil {
		h.Log.Errorf("查询分组失败: %v", err)
	}
	td := &TableGroupData{PwdMap: map[int]string{}, LinkCount: map[int]int{}}
	if list != nil {
		td.Total = list.Total
		td.Groups = list.Groups
		td.PwdMap = list.PwdMap
		td.LinkCount = list.LinkCount
	}

	data := h.newData(c, "", "table_group", "group")
	data.Page = td
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.Engine.Render(c.Writer, "table_group.html", data); err != nil {
		h.Log.Errorf("渲染分组列表失败: %v", err)
		c.String(http.StatusInternalServerError, "渲染失败")
	}
}
