// Security Roles Management JavaScript
class RoleManagement {
    constructor() {
        this.roles = [];
        this.filteredRoles = [];
        this.permissions = [];
        this.currentEditRoleId = null;
        this.currentPermissionsRoleId = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadRoles();
        this.loadPermissions();
        this.updateStats();
    }

    setupEventListeners() {
        // Search functionality
        const searchInput = document.getElementById('roleSearch');
        if (searchInput) {
            searchInput.addEventListener('input', () => this.filterRoles());
        }

        // Filter functionality
        const statusFilter = document.getElementById('statusFilter');
        if (statusFilter) {
            statusFilter.addEventListener('change', () => this.filterRoles());
        }

        // Modal close on outside click and Escape key
        this.setupModalEvents();
    }

    setupModalEvents() {
        // Close modals when clicking outside or pressing Escape
        document.addEventListener('click', (e) => {
            const modals = ['roleModal', 'roleDetailsModal', 'permissionsModal', 'confirmModal'];
            modals.forEach(modalId => {
                const modal = document.getElementById(modalId);
                if (e.target === modal) {
                    this.closeModal(modalId);
                }
            });
        });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeAllModals();
            }
        });
    }

    async loadRoles() {
        try {
            const response = await fetch('/api/security/roles');
            if (!response.ok) {
                throw new Error('HTTP error! status: ' + response.status);
            }
            
            const data = await response.json();
            this.roles = data.roles || [];
            this.filteredRoles = [...this.roles];
            this.renderRoles();
            this.updateStats();
        } catch (error) {
            console.error('Failed to load roles:', error);
            this.showError('Failed to load roles. Please try again.');
            this.roles = [];
            this.filteredRoles = [];
            this.renderRoles();
        }
    }

    async loadPermissions() {
        try {
            const response = await fetch('/security/api/permissions/definitions');
            if (!response.ok) {
                throw new Error('Failed to load permissions');
            }
            const data = await response.json();
            this.permissions = data.permissions || [];
            console.log('Loaded permissions:', this.permissions.length);
            this.populatePermissionCategories();
        } catch (error) {
            console.error('Failed to load permissions:', error);
            this.permissions = [];
        }
    }

    populatePermissionCategories() {
        const categoryFilter = document.getElementById('permissionCategoryFilter');
        if (!categoryFilter) return;

        const categories = [...new Set(this.permissions.map(p => p.category))];
        categoryFilter.innerHTML = '<option value="">All Categories</option>';
        categories.forEach(category => {
            categoryFilter.innerHTML += `<option value="${category}">${category}</option>`;
        });
    }

    filterRoles() {
        const searchTerm = document.getElementById('roleSearch')?.value.toLowerCase() || '';
        const statusFilter = document.getElementById('statusFilter')?.value || '';

        this.filteredRoles = this.roles.filter(role => {
            const matchesSearch = role.name.toLowerCase().includes(searchTerm) ||
                                (role.description && role.description.toLowerCase().includes(searchTerm));
            const matchesStatus = !statusFilter || 
                                (statusFilter === 'active' && role.isActive) ||
                                (statusFilter === 'inactive' && !role.isActive);
            
            return matchesSearch && matchesStatus;
        });

        this.renderRoles();
    }

    renderRoles() {
        const container = document.getElementById('rolesGrid');
        if (!container) return;

        if (this.filteredRoles.length === 0) {
            container.innerHTML = `
                <div class="empty-state fade-in">
                    <i class="bi bi-shield-lock"></i>
                    <h3>No Roles Found</h3>
                    <p>Get started by creating your first security role.</p>
                    <button class="rc-btn rc-btn-primary add-role-btn" onclick="showCreateRoleModal()">
                        <i class="bi bi-plus-circle"></i> Create First Role
                    </button>
                </div>
            `;
            return;
        }

        container.innerHTML = this.filteredRoles.map(role => {
            const roleIcon = role.name.charAt(0).toUpperCase() + (role.name.length > 1 ? role.name.charAt(1).toUpperCase() : '');
            const userCount = role.userRoles ? role.userRoles.length : 0;
            let permissionCount = 0;
            
            // Parse permissions if it's a JSON string
            if (role.permissions) {
                try {
                    const perms = typeof role.permissions === 'string' ? JSON.parse(role.permissions) : role.permissions;
                    permissionCount = Array.isArray(perms) ? perms.length : 0;
                } catch (e) {
                    permissionCount = 0;
                }
            }
            
            return `
                <div class="role-card" data-role-id="${role.roleID}">
                    <div class="role-icon">
                        ${roleIcon}
                    </div>
                    <div class="role-info">
                        <h3>${role.displayName || role.name}</h3>
                        <div class="role-description">${role.description || 'No description available'}</div>
                        <div class="role-meta">
                            <div class="role-meta-item">
                                <i class="bi bi-people"></i>
                                <span>${userCount} users assigned</span>
                            </div>
                            <div class="role-meta-item">
                                <i class="bi bi-key"></i>
                                <span>${permissionCount} permissions</span>
                            </div>
                            <div class="role-meta-item">
                                <i class="bi bi-calendar-plus"></i>
                                <span>Created ${role.createdAt ? new Date(role.createdAt).toLocaleDateString() : 'N/A'}</span>
                            </div>
                            <div class="role-meta-item">
                                <i class="bi bi-shield"></i>
                                <span class="status-badge ${role.isActive ? 'active' : 'inactive'}">
                                    <i class="bi bi-${role.isActive ? 'check-circle' : 'x-circle'}"></i>
                                    ${role.isActive ? 'Active' : 'Inactive'}
                                </span>
                            </div>
                        </div>
                    </div>
                    <div class="role-actions">
                        <button class="rc-btn rc-btn-sm rc-btn-ghost" onclick="viewRole(${role.roleID})">
                            <i class="bi bi-eye"></i>
                        </button>
                        <button class="rc-btn rc-btn-sm rc-btn-ghost" onclick="editRole(${role.roleID})">
                            <i class="bi bi-pencil"></i>
                        </button>
                        <button class="rc-btn rc-btn-sm rc-btn-ghost" onclick="managePermissions(${role.roleID})">
                            <i class="bi bi-key"></i>
                        </button>
                        <button class="rc-btn rc-btn-sm rc-btn-ghost" onclick="manageRoleUsers(${role.roleID})">
                            <i class="bi bi-people"></i>
                        </button>
                        <button class="rc-btn rc-btn-sm rc-btn-ghost" onclick="deleteRole(${role.roleID})">
                            <i class="bi bi-trash"></i>
                        </button>
                    </div>
                </div>
            `;
        }).join('');
        
        // Add fade-in animation to cards
        container.querySelectorAll('.role-card').forEach((card, index) => {
            card.style.animation = `fadeIn 0.5s ease-out ${index * 0.1}s both`;
        });
    }

    updateStats() {
        const totalRolesCount = this.roles.length;
        const activeUsersCount = this.roles.reduce((sum, role) => sum + (role.userRoles ? role.userRoles.length : 0), 0);
        const permissionsCount = this.roles.reduce((sum, role) => {
            if (role.permissions) {
                try {
                    const perms = typeof role.permissions === 'string' ? JSON.parse(role.permissions) : role.permissions;
                    return sum + (Array.isArray(perms) ? perms.length : 0);
                } catch (e) {
                    return sum;
                }
            }
            return sum;
        }, 0);
        
        const totalRolesEl = document.getElementById('totalRoles');
        const activeUsersEl = document.getElementById('activeUsers');
        const permissionsEl = document.getElementById('permissions');
        
        if (totalRolesEl) totalRolesEl.textContent = totalRolesCount;
        if (activeUsersEl) activeUsersEl.textContent = activeUsersCount;
        if (permissionsEl) permissionsEl.textContent = permissionsCount;
    }

    showError(message) {
        console.error('Role Management Error:', message);
        const errorDiv = document.createElement('div');
        errorDiv.className = 'rc-alert rc-alert-error';
        errorDiv.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: var(--error);
            color: white;
            padding: var(--space-md) var(--space-lg);
            border-radius: var(--radius-md);
            box-shadow: var(--shadow-lg);
            z-index: 1001;
            animation: slideInRight 0.3s ease-out;
        `;
        errorDiv.innerHTML = `<i class="bi bi-exclamation-triangle"></i> ${message}`;
        document.body.appendChild(errorDiv);
        
        setTimeout(() => errorDiv.remove(), 5000);
    }

    showSuccess(message) {
        const successDiv = document.createElement('div');
        successDiv.className = 'rc-alert rc-alert-success';
        successDiv.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: var(--success);
            color: white;
            padding: var(--space-md) var(--space-lg);
            border-radius: var(--radius-md);
            box-shadow: var(--shadow-lg);
            z-index: 1001;
            animation: slideInRight 0.3s ease-out;
        `;
        successDiv.innerHTML = `<i class="bi bi-check-circle"></i> ${message}`;
        document.body.appendChild(successDiv);
        
        setTimeout(() => successDiv.remove(), 3000);
    }

    openModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.add('active');
        }
    }

    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('active');
        }
    }

    closeAllModals() {
        const modals = ['roleModal', 'roleDetailsModal', 'permissionsModal', 'confirmModal'];
        modals.forEach(modalId => this.closeModal(modalId));
    }
}

// Global functions for role actions
function viewRole(roleId) {
    const role = window.roleManagement.roles.find(r => r.roleID === roleId);
    if (!role) {
        window.roleManagement.showError('Role not found');
        return;
    }

    let permissions = [];
    try {
        permissions = typeof role.permissions === 'string' ? JSON.parse(role.permissions) : role.permissions || [];
    } catch (e) {
        permissions = [];
    }

    const content = `
        <div style="background: linear-gradient(135deg, var(--primary-600) 0%, var(--primary-700) 100%); padding: var(--space-lg); color: white; border-radius: var(--radius-lg) var(--radius-lg) 0 0; margin: calc(var(--space-lg) * -1) calc(var(--space-lg) * -1) var(--space-lg) calc(var(--space-lg) * -1);">
            <div style="display: flex; align-items: center; gap: var(--space-md);">
                <div style="width: 60px; height: 60px; border-radius: 50%; background: linear-gradient(135deg, var(--primary-500), var(--accent-electric)); display: flex; align-items: center; justify-content: center; color: white; font-weight: 700; font-size: 1.5rem;">
                    ${role.name.charAt(0).toUpperCase()}${role.name.length > 1 ? role.name.charAt(1).toUpperCase() : ''}
                </div>
                <div>
                    <h3 style="margin: 0; font-size: 1.25rem; font-weight: 700;">${role.displayName || role.name}</h3>
                    <p style="margin: 0; opacity: 0.9; font-family: var(--font-mono);">${role.name}</p>
                </div>
                <div style="margin-left: auto;">
                    ${role.isActive ? 
                        '<span style="background: rgba(34, 197, 94, 0.2); color: white; padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); font-size: 0.75rem; font-weight: 600;"><i class="bi bi-check-circle"></i> Active</span>' :
                        '<span style="background: rgba(107, 114, 128, 0.2); color: rgba(255,255,255,0.7); padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); font-size: 0.75rem; font-weight: 600;"><i class="bi bi-x-circle"></i> Inactive</span>'
                    }
                </div>
            </div>
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-xl); margin-top: var(--space-lg);">
            <div>
                <h4 style="color: var(--text-primary); margin: 0 0 var(--space-md); display: flex; align-items: center; gap: var(--space-sm);"><i class="bi bi-info-circle"></i> Role Information</h4>
                <div style="display: flex; flex-direction: column; gap: var(--space-sm);">
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0; border-bottom: 1px solid var(--surface-3);">
                        <span style="color: var(--text-secondary);">Role ID:</span>
                        <span style="font-family: var(--font-mono); color: var(--text-primary);">${role.roleID}</span>
                    </div>
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0; border-bottom: 1px solid var(--surface-3);">
                        <span style="color: var(--text-secondary);">Display Name:</span>
                        <span style="color: var(--text-primary);">${role.displayName || 'N/A'}</span>
                    </div>
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0; border-bottom: 1px solid var(--surface-3);">
                        <span style="color: var(--text-secondary);">Description:</span>
                        <span style="color: var(--text-primary); max-width: 200px; text-align: right;">${role.description || 'No description'}</span>
                    </div>
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0;">
                        <span style="color: var(--text-secondary);">System Role:</span>
                        <span style="color: var(--text-primary);">${role.isSystemRole ? 'Yes' : 'No'}</span>
                    </div>
                </div>
            </div>
            <div>
                <h4 style="color: var(--text-primary); margin: 0 0 var(--space-md); display: flex; align-items: center; gap: var(--space-sm);"><i class="bi bi-clock"></i> Timeline</h4>
                <div style="display: flex; flex-direction: column; gap: var(--space-sm);">
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0; border-bottom: 1px solid var(--surface-3);">
                        <span style="color: var(--text-secondary);">Created:</span>
                        <span style="font-family: var(--font-mono); color: var(--text-primary); font-size: 0.875rem;">${new Date(role.createdAt).toLocaleDateString()}</span>
                    </div>
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0; border-bottom: 1px solid var(--surface-3);">
                        <span style="color: var(--text-secondary);">Updated:</span>
                        <span style="font-family: var(--font-mono); color: var(--text-primary); font-size: 0.875rem;">${new Date(role.updatedAt).toLocaleDateString()}</span>
                    </div>
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0; border-bottom: 1px solid var(--surface-3);">
                        <span style="color: var(--text-secondary);">Permissions:</span>
                        <span style="color: var(--text-primary);">${permissions.length} assigned</span>
                    </div>
                    <div style="display: flex; justify-content: space-between; padding: var(--space-xs) 0;">
                        <span style="color: var(--text-secondary);">Users:</span>
                        <span style="color: var(--text-primary);">${role.userRoles ? role.userRoles.length : 0} assigned</span>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    document.getElementById('roleDetailsContent').innerHTML = content;
    document.getElementById('editFromDetailsBtn').onclick = () => {
        closeRoleDetailsModal();
        editRole(roleId);
    };
    window.roleManagement.openModal('roleDetailsModal');
}

function editRole(roleId) {
    const role = window.roleManagement.roles.find(r => r.roleID === roleId);
    if (!role) {
        window.roleManagement.showError('Role not found');
        return;
    }

    window.roleManagement.currentEditRoleId = roleId;
    document.getElementById('roleModalTitle').textContent = 'Edit Role';
    document.getElementById('roleSubmitBtn').textContent = 'Update Role';
    
    // Populate form
    document.getElementById('roleName').value = role.name || '';
    document.getElementById('roleDisplayName').value = role.displayName || '';
    document.getElementById('roleDescription').value = role.description || '';
    document.getElementById('roleIsActive').checked = role.isActive;
    
    window.roleManagement.openModal('roleModal');
}

function managePermissions(roleId) {
    const role = window.roleManagement.roles.find(r => r.roleID === roleId);
    if (!role) {
        window.roleManagement.showError('Role not found');
        return;
    }

    window.roleManagement.currentPermissionsRoleId = roleId;
    document.getElementById('permissionsModalTitle').textContent = `Manage Permissions - ${role.displayName || role.name}`;
    
    // Load and render permissions
    renderPermissions(role);
    window.roleManagement.openModal('permissionsModal');
}

function renderPermissions(role) {
    const container = document.getElementById('permissionsContainer');
    if (!container) return;

    console.log('renderPermissions called for role:', role.name);
    console.log('Available permissions:', window.roleManagement.permissions.length);

    let rolePermissions = [];
    try {
        rolePermissions = typeof role.permissions === 'string' ? JSON.parse(role.permissions) : role.permissions || [];
    } catch (e) {
        rolePermissions = [];
    }

    // Group permissions by category
    const permissionsByCategory = {};
    window.roleManagement.permissions.forEach(permission => {
        if (!permissionsByCategory[permission.category]) {
            permissionsByCategory[permission.category] = [];
        }
        permissionsByCategory[permission.category].push(permission);
    });

    console.log('Permissions by category:', Object.keys(permissionsByCategory));

    let html = '';
    Object.keys(permissionsByCategory).forEach(category => {
        html += `
            <div class="permission-category">
                <div class="permission-category-title">${category}</div>
                <div class="permission-grid">
        `;
        
        permissionsByCategory[category].forEach(permission => {
            const isChecked = rolePermissions.includes(permission.code);
            html += `
                <div class="permission-item">
                    <input type="checkbox" id="perm_${permission.code}" value="${permission.code}" ${isChecked ? 'checked' : ''}>
                    <div class="permission-info">
                        <h4>${permission.name}</h4>
                        <p>${permission.description}</p>
                    </div>
                </div>
            `;
        });
        
        html += `
                </div>
            </div>
        `;
    });

    container.innerHTML = html;

    // Setup permission search
    const searchInput = document.getElementById('permissionSearch');
    const categoryFilter = document.getElementById('permissionCategoryFilter');
    
    const filterPermissions = () => {
        const searchTerm = searchInput.value.toLowerCase();
        const selectedCategory = categoryFilter.value;
        
        container.querySelectorAll('.permission-category').forEach(catEl => {
            const categoryTitle = catEl.querySelector('.permission-category-title').textContent;
            let hasVisibleItems = false;
            
            if (!selectedCategory || selectedCategory === categoryTitle) {
                catEl.style.display = 'block';
                
                catEl.querySelectorAll('.permission-item').forEach(item => {
                    const info = item.querySelector('.permission-info');
                    const name = info.querySelector('h4').textContent.toLowerCase();
                    const desc = info.querySelector('p').textContent.toLowerCase();
                    
                    if (!searchTerm || name.includes(searchTerm) || desc.includes(searchTerm)) {
                        item.style.display = 'flex';
                        hasVisibleItems = true;
                    } else {
                        item.style.display = 'none';
                    }
                });
                
                if (!hasVisibleItems) {
                    catEl.style.display = 'none';
                }
            } else {
                catEl.style.display = 'none';
            }
        });
    };
    
    searchInput.addEventListener('input', filterPermissions);
    categoryFilter.addEventListener('change', filterPermissions);
}

function deleteRole(roleId) {
    const role = window.roleManagement.roles.find(r => r.roleID === roleId);
    const roleName = role ? (role.displayName || role.name) : 'Unknown Role';
    
    showConfirmation(
        'Delete Role',
        `Are you sure you want to delete the role "${roleName}"? This action cannot be undone and will remove all associated permissions.`,
        async () => {
            try {
                const response = await fetch(`/api/security/roles/${roleId}`, {
                    method: 'DELETE',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (!response.ok) {
                    throw new Error('Failed to delete role');
                }
                
                window.roleManagement.showSuccess(`Role "${roleName}" deleted successfully`);
                window.roleManagement.loadRoles(); // Reload roles
                
            } catch (error) {
                console.error('Delete role error:', error);
                window.roleManagement.showError('Failed to delete role. Please try again.');
            }
        }
    );
}

function showCreateRoleModal() {
    window.roleManagement.currentEditRoleId = null;
    document.getElementById('roleModalTitle').textContent = 'Create Role';
    document.getElementById('roleSubmitBtn').textContent = 'Create Role';
    
    // Reset form
    document.getElementById('roleForm').reset();
    document.getElementById('roleIsActive').checked = true;
    
    window.roleManagement.openModal('roleModal');
}

function closeRoleModal() {
    window.roleManagement.closeModal('roleModal');
    window.roleManagement.currentEditRoleId = null;
}

function closeRoleDetailsModal() {
    window.roleManagement.closeModal('roleDetailsModal');
}

function closePermissionsModal() {
    window.roleManagement.closeModal('permissionsModal');
    window.roleManagement.currentPermissionsRoleId = null;
}

function closeConfirmModal() {
    window.roleManagement.closeModal('confirmModal');
}

function showConfirmation(title, message, onConfirm) {
    document.getElementById('confirmModalTitle').textContent = title;
    document.getElementById('confirmModalMessage').textContent = message;
    
    // Remove old event listeners
    const confirmBtn = document.getElementById('confirmBtn');
    const newConfirmBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);
    
    // Add new event listener
    newConfirmBtn.addEventListener('click', () => {
        onConfirm();
        closeConfirmModal();
    });
    
    window.roleManagement.openModal('confirmModal');
}

async function submitRole() {
    const form = document.getElementById('roleForm');
    const formData = new FormData(form);
    
    const roleData = {
        name: formData.get('name'),
        displayName: formData.get('displayName'),
        description: formData.get('description'),
        isActive: formData.get('isActive') === 'on'
    };
    
    // Validation
    if (!roleData.name.trim()) {
        window.roleManagement.showError('Role name is required');
        return;
    }
    
    if (!roleData.displayName.trim()) {
        window.roleManagement.showError('Display name is required');
        return;
    }
    
    try {
        const isEdit = window.roleManagement.currentEditRoleId !== null;
        const url = isEdit ? `/api/security/roles/${window.roleManagement.currentEditRoleId}` : '/api/security/roles';
        const method = isEdit ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(roleData)
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to save role');
        }
        
        window.roleManagement.showSuccess(`Role ${isEdit ? 'updated' : 'created'} successfully`);
        closeRoleModal();
        window.roleManagement.loadRoles(); // Reload roles
        
    } catch (error) {
        console.error('Submit role error:', error);
        window.roleManagement.showError(error.message || 'Failed to save role. Please try again.');
    }
}

async function savePermissions() {
    if (!window.roleManagement.currentPermissionsRoleId) {
        window.roleManagement.showError('No role selected for permissions update');
        return;
    }
    
    // Get selected permissions
    const selectedPermissions = [];
    document.querySelectorAll('#permissionsContainer input[type="checkbox"]:checked').forEach(checkbox => {
        selectedPermissions.push(checkbox.value);
    });
    
    try {
        const response = await fetch(`/api/security/roles/${window.roleManagement.currentPermissionsRoleId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                permissions: selectedPermissions
            })
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to update permissions');
        }
        
        window.roleManagement.showSuccess('Permissions updated successfully');
        closePermissionsModal();
        window.roleManagement.loadRoles(); // Reload roles
        
    } catch (error) {
        console.error('Save permissions error:', error);
        window.roleManagement.showError(error.message || 'Failed to save permissions. Please try again.');
    }
}

// Role User Management
let currentRoleUsersId = null;
let roleUsers = [];
let availableUsers = [];

function manageRoleUsers(roleId) {
    const role = window.roleManagement.roles.find(r => r.roleID === roleId);
    if (!role) {
        window.roleManagement.showError('Role not found');
        return;
    }

    currentRoleUsersId = roleId;
    document.getElementById('roleUsersModalTitle').textContent = `Manage Users - ${role.displayName || role.name}`;
    
    loadRoleUsers(roleId);
    loadAvailableUsersForRole(roleId);
    
    openRoleUsersModal();
}

async function loadRoleUsers(roleId) {
    try {
        // Get all users and filter those with this role
        const usersResponse = await fetch('/api/v1/security/auth/users');
        if (!usersResponse.ok) {
            throw new Error('Failed to load users');
        }
        
        const usersData = await usersResponse.json();
        const allUsers = usersData.users || [];
        
        // For each user, check if they have this role
        roleUsers = [];
        
        for (const user of allUsers) {
            try {
                const userRolesResponse = await fetch(`/security/api/users/${user.userID}/roles`);
                if (userRolesResponse.ok) {
                    const userRolesData = await userRolesResponse.json();
                    const userRoles = userRolesData.userRoles || [];
                    
                    const hasRole = userRoles.some(ur => ur.roleID === roleId || (ur.role && ur.role.roleID === roleId));
                    if (hasRole) {
                        const userRoleData = userRoles.find(ur => ur.roleID === roleId || (ur.role && ur.role.roleID === roleId));
                        roleUsers.push({
                            ...user,
                            userRoleData: userRoleData
                        });
                    }
                }
            } catch (error) {
                console.error(`Error checking roles for user ${user.userID}:`, error);
            }
        }
        
        renderRoleUsers();
    } catch (error) {
        console.error('Error loading role users:', error);
        document.getElementById('currentUsersList').innerHTML = '<div class="empty-users">Failed to load users</div>';
    }
}

async function loadAvailableUsersForRole(roleId) {
    try {
        const response = await fetch('/api/v1/security/auth/users');
        if (!response.ok) {
            throw new Error('Failed to load available users');
        }
        
        const data = await response.json();
        const allUsers = data.users || [];
        
        // Filter out users who already have this role
        const assignedUserIds = roleUsers.map(ru => ru.userID);
        availableUsers = allUsers.filter(user => !assignedUserIds.includes(user.userID));
        
        renderAvailableUsersForRole();
    } catch (error) {
        console.error('Error loading available users:', error);
        document.getElementById('availableUsersList').innerHTML = '<div class="empty-users">Failed to load available users</div>';
    }
}

function renderRoleUsers() {
    const container = document.getElementById('currentUsersList');
    
    if (!roleUsers || roleUsers.length === 0) {
        container.innerHTML = '<div class="empty-users">No users assigned to this role</div>';
        return;
    }
    
    container.innerHTML = roleUsers.map(user => {
        const firstName = user.firstName || '';
        const lastName = user.lastName || '';
        const avatar = firstName.charAt(0) + lastName.charAt(0) || user.username.charAt(0).toUpperCase();
        const fullName = `${firstName} ${lastName}`.trim() || user.username;
        
        const expiryText = user.userRoleData?.expiresAt ? 
            `<span style="font-size: 0.75rem; color: var(--text-muted);">Expires: ${new Date(user.userRoleData.expiresAt).toLocaleDateString()}</span>` : 
            '<span style="font-size: 0.75rem; color: var(--text-muted);">No expiry</span>';
        
        return `
            <div class="user-item">
                <div class="user-item-info">
                    <div class="user-item-avatar">${avatar}</div>
                    <div class="user-item-details">
                        <div class="user-item-name">${fullName}</div>
                        <div class="user-item-username">@${user.username}</div>
                        <div class="user-item-meta">
                            <span style="font-size: 0.75rem; color: var(--text-muted);">
                                <i class="bi bi-envelope"></i> ${user.email}
                            </span>
                            ${expiryText}
                        </div>
                    </div>
                </div>
                <div class="user-item-actions">
                    <button class="rc-btn rc-btn-sm rc-btn-ghost" onclick="revokeRoleFromUser(${user.userID}, '${fullName}')" title="Remove Role">
                        <i class="bi bi-dash-circle"></i>
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

function renderAvailableUsersForRole() {
    const container = document.getElementById('availableUsersList');
    
    if (!availableUsers || availableUsers.length === 0) {
        container.innerHTML = '<div class="empty-users">No additional users available</div>';
        return;
    }
    
    container.innerHTML = availableUsers.map(user => {
        const firstName = user.firstName || '';
        const lastName = user.lastName || '';
        const avatar = firstName.charAt(0) + lastName.charAt(0) || user.username.charAt(0).toUpperCase();
        const fullName = `${firstName} ${lastName}`.trim() || user.username;
        
        return `
            <div class="user-item">
                <div class="user-item-info">
                    <div class="user-item-avatar">${avatar}</div>
                    <div class="user-item-details">
                        <div class="user-item-name">${fullName}</div>
                        <div class="user-item-username">@${user.username}</div>
                        <div class="user-item-meta">
                            <span style="font-size: 0.75rem; color: var(--text-muted);">
                                <i class="bi bi-envelope"></i> ${user.email}
                            </span>
                            <span class="status-badge ${user.isActive ? 'active' : 'inactive'}">
                                <i class="bi bi-${user.isActive ? 'check-circle' : 'x-circle'}"></i>
                                ${user.isActive ? 'Active' : 'Inactive'}
                            </span>
                        </div>
                    </div>
                </div>
                <div class="user-item-actions">
                    <button class="rc-btn rc-btn-sm rc-btn-primary" onclick="assignRoleToUser(${user.userID}, '${fullName}')" title="Assign Role">
                        <i class="bi bi-plus-circle"></i> Assign
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

async function assignRoleToUser(userId, userName) {
    try {
        const response = await fetch(`/security/api/users/${userId}/roles`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                roleId: currentRoleUsersId
            })
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to assign role');
        }
        
        window.roleManagement.showSuccess(`Role assigned to "${userName}" successfully`);
        
        // Refresh the user lists
        await loadRoleUsers(currentRoleUsersId);
        await loadAvailableUsersForRole(currentRoleUsersId);
        
    } catch (error) {
        console.error('Error assigning role:', error);
        window.roleManagement.showError('Error: ' + error.message);
    }
}

function revokeRoleFromUser(userId, userName) {
    showConfirmation(
        'Revoke Role',
        `Are you sure you want to remove this role from "${userName}"?`,
        async () => {
            try {
                const response = await fetch(`/security/api/users/${userId}/roles/${currentRoleUsersId}`, {
                    method: 'DELETE',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Failed to revoke role');
                }
                
                window.roleManagement.showSuccess(`Role removed from "${userName}" successfully`);
                
                // Refresh the user lists
                await loadRoleUsers(currentRoleUsersId);
                await loadAvailableUsersForRole(currentRoleUsersId);
                
            } catch (error) {
                console.error('Error revoking role:', error);
                window.roleManagement.showError('Error: ' + error.message);
            }
        }
    );
}

function openRoleUsersModal() {
    const modal = document.getElementById('roleUsersModal');
    if (modal) {
        modal.classList.add('active');
    }
}

function closeRoleUsersModal() {
    const modal = document.getElementById('roleUsersModal');
    if (modal) {
        modal.classList.remove('active');
    }
    currentRoleUsersId = null;
    roleUsers = [];
    availableUsers = [];
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    window.roleManagement = new RoleManagement();
    
    // Add close functionality for role users modal
    document.addEventListener('click', (e) => {
        const roleUsersModal = document.getElementById('roleUsersModal');
        if (e.target === roleUsersModal) {
            closeRoleUsersModal();
        }
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const roleUsersModal = document.getElementById('roleUsersModal');
            if (roleUsersModal && roleUsersModal.classList.contains('active')) {
                closeRoleUsersModal();
            }
        }
    });
});