package admin

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
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

// GroupPage 分组管理 group.php
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
			id := postInt(c, "group_id")
			if err := h.Store.DB().Where("group_id = ?", id).Delete(&models.Link{}).Error; err != nil {
				h.Log.Errorf("删除分组链接失败: %v", err)
			}
			if err := h.Store.DB().Where("group_id = ?", id).Delete(&models.Group{}).Error; err != nil {
				h.Log.Errorf("删除分组失败: %v", err)
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

	var pwds []models.Pwd
	if err := h.Store.DB().Find(&pwds).Error; err != nil {
		h.Log.Errorf("查询加密组失败: %v", err)
	}
	gp := &groupPage{Set: set, Pwds: pwds}
	if set == "edit" {
		id := queryInt(c, "id")
		var row models.Group
		if err := h.Store.DB().Where("group_id = ?", id).First(&row).Error; err != nil {
			h.renderHTML(c, alert("提示", "当前记录不存在！", "./group.php"))
			return
		}
		gp.Edit = &row
		gp.IconType, gp.IconVal = groupIconType(row.GroupIcon)
	}
	data := h.newData(c, "分组管理", "group.php", "group")
	data.Page = gp
	h.render(c, "group.html", data)
}

// groupIconType 判断图标类型并提取图片地址
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

// normalizeGroupIcon 纯 URL 自动包裹 img 标签
func normalizeGroupIcon(icon string) string {
	if icon != "" && !strings.HasPrefix(icon, "<") {
		return `<img src="` + icon + `" class="lylme-group-img" />`
	}
	return icon
}

func (h *Handler) groupAdd(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("group_name"))
	icon := normalizeGroupIcon(strings.TrimSpace(c.PostForm("group_icon")))
	pwd := postInt(c, "group_pwd")
	if name == "" {
		h.renderHTML(c, notify("保存错误,请确保带星号的都不为空！", "danger", 3000))
		return
	}
	var maxOrder int
	_ = h.Store.DB().Model(&models.Group{}).Select("COALESCE(MAX(group_order),0)").Scan(&maxOrder).Error
	rec := models.Group{GroupName: name, GroupIcon: icon, GroupOrder: maxOrder + 1, GroupPwd: pwd, GroupStatus: 1}
	if err := h.Store.DB().Create(&rec).Error; err != nil {
		h.Log.Errorf("新增分组失败: %v", err)
		h.renderHTML(c, notify("添加分组失败", "danger", 3000))
		return
	}
	h.renderHTML(c, alert("成功", "添加分组 "+htmlEscape(name)+" 成功！", "./group.php"))
}

func (h *Handler) groupEdit(c *gin.Context) {
	id := queryInt(c, "id")
	var row models.Group
	if err := h.Store.DB().Where("group_id = ?", id).First(&row).Error; err != nil {
		h.renderHTML(c, notify("当前记录不存在！", "danger", 3000))
		return
	}
	name := strings.TrimSpace(c.PostForm("group_name"))
	icon := normalizeGroupIcon(strings.TrimSpace(c.PostForm("group_icon")))
	pwd := postInt(c, "group_pwd")
	if name == "" {
		h.renderHTML(c, notify("保存错误,请确保带星号的都不为空！", "danger", 3000))
		return
	}
	row.GroupName, row.GroupIcon, row.GroupPwd = name, icon, pwd
	if err := h.Store.DB().Save(&row).Error; err != nil {
		h.Log.Errorf("修改分组失败: %v", err)
		h.renderHTML(c, notify("修改分组失败", "danger", 3000))
		return
	}
	h.renderHTML(c, alert("成功", "修改分组 "+htmlEscape(name)+" 成功！", "./group.php"))
}

func (h *Handler) setGroupStatus(c *gin.Context, status int) {
	id := postInt(c, "group_id")
	if err := h.Store.DB().Model(&models.Group{}).Where("group_id = ?", id).Update("group_status", status).Error; err != nil {
		h.Log.Errorf("更新分组状态失败: %v", err)
	}
	c.Status(http.StatusOK)
}

func (h *Handler) groupSort(c *gin.Context) {
	for i, s := range postStrings(c, "groups") {
		id, err := atoi(s)
		if err != nil {
			continue
		}
		if err := h.Store.DB().Model(&models.Group{}).Where("group_id = ?", id).Update("group_order", i).Error; err != nil {
			h.Log.Errorf("更新分组排序失败: %v", err)
		}
	}
	c.Status(http.StatusOK)
}

// TableGroup table_group.php 片段
func (h *Handler) TableGroup(c *gin.Context) {
	td := &TableGroupData{PwdMap: map[int]string{}, LinkCount: map[int]int{}}
	if err := h.Store.DB().Order("group_order ASC").Find(&td.Groups).Error; err != nil {
		h.Log.Errorf("查询分组失败: %v", err)
	}
	td.Total = int64(len(td.Groups))

	var pwds []models.Pwd
	if err := h.Store.DB().Find(&pwds).Error; err == nil {
		for _, p := range pwds {
			td.PwdMap[p.PwdID] = p.PwdKey
		}
	}
	type row struct {
		GroupID int
		Cnt     int
	}
	var rows []row
	if err := h.Store.DB().Model(&models.Link{}).
		Select("group_id, COUNT(id) AS cnt").Group("group_id").Scan(&rows).Error; err == nil {
		for _, r := range rows {
			td.LinkCount[r.GroupID] = r.Cnt
		}
	}

	data := h.newData(c, "", "table_group.php", "group")
	data.Page = td
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.Engine.Render(c.Writer, "table_group.html", data); err != nil {
		h.Log.Errorf("渲染分组列表失败: %v", err)
		c.String(http.StatusInternalServerError, "渲染失败")
	}
}
