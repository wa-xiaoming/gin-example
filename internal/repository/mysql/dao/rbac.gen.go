package dao

import (
	"context"
	
	"gin-example/internal/repository/mysql/model"
	"gorm.io/gorm"
)

type IRBAC interface {
	// GetUserRoles 获取用户角色
	GetUserRoles(ctx context.Context, userID int32) ([]*model.Role, error)
	
	// GetUserPermissions 获取用户权限
	GetUserPermissions(ctx context.Context, userID int32) ([]*model.Permission, error)
	
	// CheckPermission 检查用户是否有特定权限
	CheckPermission(ctx context.Context, userID int32, resource, action string) (bool, error)
	
	// CreateRole 创建角色
	CreateRole(ctx context.Context, role *model.Role) error
	
	// CreatePermission 创建权限
	CreatePermission(ctx context.Context, permission *model.Permission) error
	
	// AssignRoleToUser 为用户分配角色
	AssignRoleToUser(ctx context.Context, userID, roleID int32) error
	
	// AssignPermissionToRole 为角色分配权限
	AssignPermissionToRole(ctx context.Context, roleID, permissionID int32) error
}

type rbac struct {
	db *gorm.DB
}

func NewRBAC(db *gorm.DB) IRBAC {
	return &rbac{db: db}
}

// GetUserRoles 获取用户角色
func (r *rbac) GetUserRoles(ctx context.Context, userID int32) ([]*model.Role, error) {
	var roles []*model.Role
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

// GetUserPermissions 获取用户权限
func (r *rbac) GetUserPermissions(ctx context.Context, userID int32) ([]*model.Permission, error) {
	var permissions []*model.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&permissions).Error
	return permissions, err
}

// CheckPermission 检查用户是否有特定权限
func (r *rbac) CheckPermission(ctx context.Context, userID int32, resource, action string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ? AND permissions.resource = ? AND permissions.action = ?", userID, resource, action).
		Count(&count).Error
	
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// CreateRole 创建角色
func (r *rbac) CreateRole(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// CreatePermission 创建权限
func (r *rbac) CreatePermission(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

// AssignRoleToUser 为用户分配角色
func (r *rbac) AssignRoleToUser(ctx context.Context, userID, roleID int32) error {
	userRole := &model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}

// AssignPermissionToRole 为角色分配权限
func (r *rbac) AssignPermissionToRole(ctx context.Context, roleID, permissionID int32) error {
	rolePermission := &model.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	return r.db.WithContext(ctx).Create(rolePermission).Error
}