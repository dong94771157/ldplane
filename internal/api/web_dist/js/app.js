// ==================== 全局工具 ====================
let token = localStorage.getItem('token') || '';
let currentPath = '/';
let dashboardTimer = null;
let terminalWs = null;

function api(url, options = {}) {
    const opts = { ...options, headers: { 'Content-Type': 'application/json', ...options.headers } };
    if (token) opts.headers['Authorization'] = 'Bearer ' + token;
    return fetch(url, opts).then(r => {
        if (r.status === 401) { doLogout(); throw new Error('未登录'); }
        return r.json();
    });
}

function toast(msg, type = 'success') {
    const el = document.getElementById('toast');
    el.textContent = msg;
    el.className = 'toast toast-' + type;
    el.style.display = 'block';
    setTimeout(() => el.style.display = 'none', 3000);
}

function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let i = 0;
    while (bytes >= 1024 && i < units.length - 1) { bytes /= 1024; i++; }
    return bytes.toFixed(1) + ' ' + units[i];
}

function formatUptime(seconds) {
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return d + '天' + h + '小时' + m + '分';
}

function escapeHtml(s) {
    return String(s).replace(/&/g,'&'+'amp;').replace(/</g,'&'+'lt;').replace(/>/g,'&'+'gt;').replace(/"/g,'&'+'quot;');
}

function showModal(html) {
    document.getElementById('modal-box').innerHTML = html;
    document.getElementById('modal-overlay').style.display = 'flex';
}
function closeModal() { document.getElementById('modal-overlay').style.display = 'none'; }
document.getElementById('modal-overlay')?.addEventListener('click', function(e) {
    if (e.target === this) closeModal();
});

// ==================== 登录/登出 ====================
function doLogin() {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    if (!username || !password) { toast('请填写用户名和密码', 'error'); return; }
    api('/api/login', { method: 'POST', body: JSON.stringify({ username, password }) })
        .then(res => {
            if (res.code === 0) {
                token = res.data.token;
                localStorage.setItem('token', token);
                showMain();
                toast('登录成功');
            } else {
                toast(res.msg || '登录失败', 'error');
            }
        })
        .catch(() => toast('网络错误', 'error'));
}

function doLogout() {
    token = '';
    localStorage.removeItem('token');
    if (dashboardTimer) { clearInterval(dashboardTimer); dashboardTimer = null; }
    if (terminalWs) { terminalWs.close(); terminalWs = null; }
    document.getElementById('main-page').style.display = 'none';
    document.getElementById('login-page').style.display = 'flex';
}

function showMain() {
    document.getElementById('login-page').style.display = 'none';
    document.getElementById('main-page').style.display = 'flex';
    switchPage('dashboard');
}

// ==================== 页面切换 ====================
function switchPage(page) {
    document.querySelectorAll('.nav-item').forEach(el => el.classList.toggle('active', el.dataset.page === page));
    document.querySelectorAll('.page').forEach(el => el.style.display = 'none');
    document.getElementById('page-' + page).style.display = 'block';
    // 停止仪表盘定时器
    if (dashboardTimer && page !== 'dashboard') { clearInterval(dashboardTimer); dashboardTimer = null; }
    // 加载对应页面数据
    const loaders = {
        dashboard: () => { loadDashboard(); dashboardTimer = setInterval(loadDashboard, 5000); },
        site: loadSites, file: () => loadFiles(currentPath), service: loadServices,
        firewall: loadFirewall, process: loadProcess, terminal: initTerminal, settings: () => {}, about: () => {}
    };
    if (loaders[page]) loaders[page]();
}

// ==================== 仪表盘 ====================
function loadDashboard() {
    api('/api/system/overview').then(res => {
        if (res.code !== 0) return;
        const d = res.data;
        const cpuPct = d.cpu.percent && d.cpu.percent.length > 0 ? d.cpu.percent[0].toFixed(1) : 0;
        document.getElementById('cpu-percent').textContent = cpuPct + '%';
        document.getElementById('cpu-bar').style.width = cpuPct + '%';
        document.getElementById('cpu-model').textContent = d.cpu.model + ' (' + d.cpu.cores + '核)';

        const memPct = d.memory.used_percent.toFixed(1);
        document.getElementById('mem-percent').textContent = memPct + '%';
        document.getElementById('mem-bar').style.width = memPct + '%';
        document.getElementById('mem-detail').textContent = formatBytes(d.memory.used) + ' / ' + formatBytes(d.memory.total);

        document.getElementById('load-val').textContent = d.load.load1.toFixed(2);
        document.getElementById('load-detail').textContent = '1分/5分/15分: ' + d.load.load1.toFixed(2) + ' / ' + d.load.load5.toFixed(2) + ' / ' + d.load.load15.toFixed(2);

        document.getElementById('uptime-val').textContent = formatUptime(d.host.uptime);
        document.getElementById('uptime-detail').textContent = d.host.hostname + ' | ' + d.host.platform;

        // 磁盘
        let diskHtml = '';
        (d.disks || []).forEach(dk => {
            const pct = dk.used_percent.toFixed(1);
            diskHtml += '<tr><td>' + escapeHtml(dk.device) + '</td><td>' + escapeHtml(dk.mountpoint) +
                '</td><td>' + dk.fstype + '</td><td>' + formatBytes(dk.total) + '</td><td>' + formatBytes(dk.used) +
                '</td><td><span class="badge ' + (pct > 90 ? 'badge-danger' : pct > 70 ? 'badge-warning' : 'badge-success') + '">' + pct + '%</span></td></tr>';
        });
        document.querySelector('#disk-table tbody').innerHTML = diskHtml;

        // 网络
        if (d.network) {
            document.getElementById('net-send').textContent = formatBytes(d.network.bytes_sent);
            document.getElementById('net-recv').textContent = formatBytes(d.network.bytes_recv);
        }
    }).catch(() => {});
}

// ==================== 网站管理 ====================
function loadSites() {
    api('/api/site/list').then(res => {
        if (res.code !== 0) return;
        let html = '';
        (res.data || []).forEach(s => {
            html += '<tr><td>' + escapeHtml(s.name) + '</td><td>' + escapeHtml(s.domain) + '</td><td>' + s.port +
                '</td><td>' + escapeHtml(s.root) + '</td><td>' + (s.ssl ? '<span class="badge badge-success">是</span>' : '否') +
                '</td><td>' + (s.enabled ? '<span class="badge badge-success">运行中</span>' : '<span class="badge badge-danger">已停止</span>') +
                '</td><td>' +
                '<button class="btn btn-sm ' + (s.enabled ? 'btn-danger' : 'btn-success') + '" onclick="toggleSite(\'' + s.name + '\',' + !s.enabled + ')">' + (s.enabled ? '停止' : '启动') + '</button> ' +
                '<button class="btn btn-sm btn-outline" onclick="editSiteConf(\'' + s.name + '\')">配置</button> ' +
                '<button class="btn btn-sm btn-danger" onclick="deleteSite(\'' + s.name + '\')">删除</button>' +
                '</td></tr>';
        });
        document.getElementById('site-tbody').innerHTML = html || '<tr><td colspan="7" style="text-align:center;color:#999">暂无网站</td></tr>';
    });
}

function showSiteModal() {
    showModal(`<h3>添加网站</h3>
    <div class="form-group"><label>名称</label><input type="text" id="site-name" placeholder="如: mysite"></div>
    <div class="form-group"><label>域名</label><input type="text" id="site-domain" placeholder="如: example.com"></div>
    <div class="form-group"><label>端口</label><input type="text" id="site-port" value="80" placeholder="如: 80"></div>
    <div class="form-group"><label>根目录</label><input type="text" id="site-root" value="/var/www/" placeholder="如: /var/www/mysite"></div>
    <div class="modal-actions">
        <button class="btn btn-outline" onclick="closeModal()">取消</button>
        <button class="btn btn-primary" onclick="createSite()">创建</button>
    </div>`);
}

function createSite() {
    const data = {
        name: document.getElementById('site-name').value,
        domain: document.getElementById('site-domain').value,
        port: document.getElementById('site-port').value,
        root: document.getElementById('site-root').value
    };
    if (!data.name || !data.domain || !data.port || !data.root) { toast('请填写完整信息', 'error'); return; }
    api('/api/site/create', { method: 'POST', body: JSON.stringify(data) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        if (res.code === 0) { closeModal(); loadSites(); }
    });
}

function toggleSite(name, enabled) {
    api('/api/site/toggle', { method: 'POST', body: JSON.stringify({ name, enabled }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadSites();
    });
}

function deleteSite(name) {
    if (!confirm('确定删除网站 ' + name + ' 吗？')) return;
    api('/api/site/delete', { method: 'POST', body: JSON.stringify({ name }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadSites();
    });
}

function editSiteConf(name) {
    api('/api/site/conf?name=' + name).then(res => {
        if (res.code !== 0) { toast(res.msg, 'error'); return; }
        showModal(`<h3>编辑配置 - ${escapeHtml(name)}</h3>
        <div class="form-group"><textarea id="site-conf" rows="20">${escapeHtml(res.data)}</textarea></div>
        <div class="modal-actions">
            <button class="btn btn-outline" onclick="closeModal()">取消</button>
            <button class="btn btn-primary" onclick="saveSiteConf('${name}')">保存</button>
        </div>`);
    });
}

function saveSiteConf(name) {
    const content = document.getElementById('site-conf').value;
    api('/api/site/conf', { method: 'POST', body: JSON.stringify({ name, content }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        if (res.code === 0) closeModal();
    });
}

// ==================== 文件管理 ====================
function loadFiles(path) {
    currentPath = path;
    api('/api/file/list?path=' + encodeURIComponent(path)).then(res => {
        if (res.code !== 0) { toast(res.msg, 'error'); return; }
        const data = res.data;
        currentPath = data.path;
        // 面包屑
        const parts = data.path.split('/').filter(Boolean);
        let breadcrumb = '<span onclick="loadFiles(\'/\')">/</span>';
        let acc = '';
        parts.forEach(p => { acc += '/' + p; breadcrumb += '<span onclick="loadFiles(\'' + acc.replace(/'/g,"\\'") + '\')">' + escapeHtml(p) + '/</span>'; });
        document.getElementById('file-path').innerHTML = breadcrumb;

        let html = '';
        if (data.path !== '/') {
            const parent = data.path.replace(/\/[^/]+\/?$/, '') || '/';
            html += '<tr><td colspan="5"><a onclick="loadFiles(\'' + parent.replace(/'/g,"\\'") + '\')" style="cursor:pointer;color:#667eea">.. 返回上级</a></td></tr>';
        }
        (data.items || []).forEach(f => {
            const p = f.path.replace(/'/g, "\\'");
            if (f.is_dir) {
                html += '<tr><td><a onclick="loadFiles(\'' + p + '\')" style="cursor:pointer;color:#667eea">📁 ' + escapeHtml(f.name) + '</a></td>' +
                    '<td>-</td><td>' + f.mode + '</td><td>' + f.mod_time + '</td>' +
                    '<td><button class="btn btn-sm btn-danger" onclick="deleteFile(\'' + p + '\')">删除</button> ' +
                    '<button class="btn btn-sm btn-outline" onclick="renameFile(\'' + p + '\')">重命名</button> ' +
                    '<button class="btn btn-sm btn-outline" onclick="compressFile(\'' + p + '\')">压缩</button></td></tr>';
            } else {
                html += '<tr><td><a onclick="editFile(\'' + p + '\')" style="cursor:pointer">📄 ' + escapeHtml(f.name) + '</a></td>' +
                    '<td>' + formatBytes(f.size) + '</td><td>' + f.mode + '</td><td>' + f.mod_time + '</td>' +
                    '<td><a class="btn btn-sm btn-outline" href="/api/file/download?path=' + encodeURIComponent(f.path) + '&token=' + token + '">下载</a> ' +
                    '<button class="btn btn-sm btn-danger" onclick="deleteFile(\'' + p + '\')">删除</button> ' +
                    '<button class="btn btn-sm btn-outline" onclick="renameFile(\'' + p + '\')">重命名</button> ' +
                    '<button class="btn btn-sm btn-outline" onclick="editFile(\'' + p + '\')">编辑</button> ' +
                    '<button class="btn btn-sm btn-outline" onclick="compressFile(\'' + p + '\')">压缩</button></td></tr>';
            }
        });
        document.getElementById('file-tbody').innerHTML = html || '<tr><td colspan="5" style="text-align:center;color:#999">空目录</td></tr>';
    });
}

function deleteFile(path) {
    if (!confirm('确定删除 ' + path + ' 吗？')) return;
    api('/api/file/delete', { method: 'POST', body: JSON.stringify({ path }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadFiles(currentPath);
    });
}

function renameFile(path) {
    const newName = prompt('请输入新名称:', path.split('/').pop());
    if (!newName) return;
    const newPath = path.substring(0, path.lastIndexOf('/') + 1) + newName;
    api('/api/file/rename', { method: 'POST', body: JSON.stringify({ old_path: path, new_path: newPath }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadFiles(currentPath);
    });
}

function compressFile(path) {
    api('/api/file/compress', { method: 'POST', body: JSON.stringify({ path }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadFiles(currentPath);
    });
}

function editFile(path) {
    api('/api/file/content?path=' + encodeURIComponent(path)).then(res => {
        if (res.code !== 0) { toast(res.msg, 'error'); return; }
        showModal(`<h3>编辑文件</h3>
        <p style="font-size:12px;color:#999;margin-bottom:10px">${escapeHtml(path)}</p>
        <div class="form-group"><textarea id="file-editor" rows="22" style="font-size:13px">${escapeHtml(res.data)}</textarea></div>
        <div class="modal-actions">
            <button class="btn btn-outline" onclick="closeModal()">取消</button>
            <button class="btn btn-primary" onclick="saveFile('${path.replace(/'/g,"\\'")}')">保存</button>
        </div>`);
    });
}

function saveFile(path) {
    const content = document.getElementById('file-editor').value;
    api('/api/file/save', { method: 'POST', body: JSON.stringify({ path, content }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        if (res.code === 0) closeModal();
    });
}

function showMkdirModal() {
    showModal(`<h3>新建文件夹</h3>
    <div class="form-group"><label>路径</label><input type="text" id="mkdir-path" value="${currentPath === '/' ? '/' : currentPath + '/'}"></div>
    <div class="modal-actions">
        <button class="btn btn-outline" onclick="closeModal()">取消</button>
        <button class="btn btn-primary" onclick="mkdir()">创建</button>
    </div>`);
}

function mkdir() {
    const path = document.getElementById('mkdir-path').value;
    if (!path) { toast('请输入路径', 'error'); return; }
    api('/api/file/mkdir', { method: 'POST', body: JSON.stringify({ path }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        if (res.code === 0) { closeModal(); loadFiles(currentPath); }
    });
}

function showUploadModal() {
    showModal(`<h3>上传文件</h3>
    <p style="font-size:12px;color:#999;margin-bottom:10px">目标目录: ${escapeHtml(currentPath)}</p>
    <div class="form-group"><input type="file" id="upload-file" multiple></div>
    <div class="modal-actions">
        <button class="btn btn-outline" onclick="closeModal()">取消</button>
        <button class="btn btn-primary" onclick="uploadFile()">上传</button>
    </div>`);
}

function uploadFile() {
    const files = document.getElementById('upload-file').files;
    if (!files.length) { toast('请选择文件', 'error'); return; }
    const formData = new FormData();
    formData.append('path', currentPath);
    for (const f of files) formData.append('file', f);
    fetch('/api/file/upload', { method: 'POST', headers: { 'Authorization': 'Bearer ' + token }, body: formData })
        .then(r => r.json()).then(res => {
            toast(res.msg, res.code === 0 ? 'success' : 'error');
            if (res.code === 0) { closeModal(); loadFiles(currentPath); }
        });
}

// ==================== 服务管理 ====================
function loadServices() {
    api('/api/service/list').then(res => {
        if (res.code !== 0) return;
        let html = '';
        (res.data || []).forEach(s => {
            const running = s.active_state === 'active';
            html += '<tr><td>' + escapeHtml(s.name) + '</td><td>' + escapeHtml(s.description) +
                '</td><td><span class="badge ' + (running ? 'badge-success' : 'badge-danger') + '">' + s.active_state + ' (' + s.sub_state + ')</span></td>' +
                '<td>' +
                '<button class="btn btn-sm btn-success" onclick="serviceAction(\'' + s.name + '\',\'start\')">启动</button> ' +
                '<button class="btn btn-sm btn-danger" onclick="serviceAction(\'' + s.name + '\',\'stop\')">停止</button> ' +
                '<button class="btn btn-sm btn-outline" onclick="serviceAction(\'' + s.name + '\',\'restart\')">重启</button> ' +
                '<button class="btn btn-sm btn-outline" onclick="serviceAction(\'' + s.name + '\',\'reload\')">重载</button>' +
                '</td></tr>';
        });
        document.getElementById('service-tbody').innerHTML = html || '<tr><td colspan="4" style="text-align:center;color:#999">暂无服务</td></tr>';
    });
}

function serviceAction(name, action) {
    api('/api/service/action', { method: 'POST', body: JSON.stringify({ name, action }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadServices();
    });
}

// ==================== 防火墙 ====================
function loadFirewall() {
    api('/api/firewall/list').then(res => {
        if (res.code !== 0) return;
        let html = '';
        (res.data || []).forEach(r => {
            html += '<tr><td>' + r.port + '</td><td>' + r.protocol + '</td><td><span class="badge badge-info">' + r.action + '</span></td>' +
                '<td><button class="btn btn-sm btn-danger" onclick="deleteFirewallRule(\'' + r.port + '\',\'' + r.protocol + '\')">删除</button></td></tr>';
        });
        document.getElementById('firewall-tbody').innerHTML = html || '<tr><td colspan="4" style="text-align:center;color:#999">暂无规则</td></tr>';
    });
}

function showFirewallModal() {
    showModal(`<h3>添加防火墙规则</h3>
    <div class="form-group"><label>端口</label><input type="number" id="fw-port" placeholder="如: 8080"></div>
    <div class="form-group"><label>协议</label><select id="fw-proto"><option value="tcp">TCP</option><option value="udp">UDP</option></select></div>
    <div class="modal-actions">
        <button class="btn btn-outline" onclick="closeModal()">取消</button>
        <button class="btn btn-primary" onclick="addFirewallRule()">添加</button>
    </div>`);
}

function addFirewallRule() {
    const port = document.getElementById('fw-port').value;
    const protocol = document.getElementById('fw-proto').value;
    if (!port) { toast('请输入端口', 'error'); return; }
    api('/api/firewall/add', { method: 'POST', body: JSON.stringify({ port, protocol }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        if (res.code === 0) { closeModal(); loadFirewall(); }
    });
}

function deleteFirewallRule(port, protocol) {
    if (!confirm('确定删除规则 ' + port + '/' + protocol + ' 吗？')) return;
    api('/api/firewall/delete', { method: 'POST', body: JSON.stringify({ port, protocol }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadFirewall();
    });
}

// ==================== 进程管理 ====================
function loadProcess() {
    api('/api/process/list').then(res => {
        if (res.code !== 0) return;
        let html = '';
        (res.data || []).forEach(p => {
            html += '<tr><td>' + p.pid + '</td><td>' + escapeHtml(p.name) + '</td><td>' + p.cpu.toFixed(1) +
                '</td><td>' + p.memory.toFixed(1) + '</td><td>' + escapeHtml(p.user) +
                '</td><td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" title="' + escapeHtml(p.command) + '">' + escapeHtml(p.command) +
                '</td><td><button class="btn btn-sm btn-danger" onclick="killProcess(' + p.pid + ')">结束</button></td></tr>';
        });
        document.getElementById('process-tbody').innerHTML = html || '<tr><td colspan="7" style="text-align:center;color:#999">暂无进程</td></tr>';
    });
}

function killProcess(pid) {
    if (!confirm('确定结束进程 PID ' + pid + ' 吗？')) return;
    api('/api/process/kill', { method: 'POST', body: JSON.stringify({ pid }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        loadProcess();
    });
}

// ==================== 终端 ====================
function initTerminal() {
    const output = document.getElementById('terminal-output');
    output.innerHTML = '<div style="color:#4ec9b0">琳东linux管理面板 Web终端 - 输入命令后按回车执行</div>\n';
    const input = document.getElementById('terminal-input');
    input.value = '';
    input.focus();
    input.onkeydown = function(e) {
        if (e.key === 'Enter') {
            const cmd = input.value.trim();
            if (!cmd) return;
            termPrint('$ ' + cmd, 'prompt');
            input.value = '';
            // 通过 WebSocket 发送命令
            if (terminalWs && terminalWs.readyState === WebSocket.OPEN) {
                terminalWs.send(cmd + '\n');
            } else {
                connectTerminal();
                setTimeout(() => { if (terminalWs && terminalWs.readyState === WebSocket.OPEN) terminalWs.send(cmd + '\n'); }, 500);
            }
        }
    };
    connectTerminal();
}

function connectTerminal() {
    if (terminalWs) { try { terminalWs.close(); } catch(e) {} terminalWs = null; }
    const wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/api/terminal/ws?token=' + token;
    terminalWs = new WebSocket(wsUrl);
    terminalWs.onmessage = function(e) {
        termPrint(e.data, 'output');
    };
    terminalWs.onclose = function() {
        termPrint('\n[连接已断开]', 'error');
    };
    terminalWs.onerror = function() {
        termPrint('[连接错误]', 'error');
    };
}

function termPrint(text, type) {
    const output = document.getElementById('terminal-output');
    const div = document.createElement('div');
    if (type === 'prompt') div.style.color = '#4ec9b0';
    else if (type === 'error') div.style.color = '#f44747';
    div.textContent = text;
    output.appendChild(div);
    output.scrollTop = output.scrollHeight;
}

// ==================== 设置 ====================
function changePassword() {
    const oldPwd = document.getElementById('old-password').value;
    const newPwd = document.getElementById('new-password').value;
    const confirmPwd = document.getElementById('confirm-password').value;
    if (!oldPwd || !newPwd) { toast('请填写完整', 'error'); return; }
    if (newPwd !== confirmPwd) { toast('两次密码不一致', 'error'); return; }
    if (newPwd.length < 6) { toast('新密码至少6位', 'error'); return; }
    api('/api/password', { method: 'POST', body: JSON.stringify({ old_password: oldPwd, new_password: newPwd }) }).then(res => {
        toast(res.msg, res.code === 0 ? 'success' : 'error');
        if (res.code === 0) {
            document.getElementById('old-password').value = '';
            document.getElementById('new-password').value = '';
            document.getElementById('confirm-password').value = '';
        }
    });
}

// ==================== 初始化 ====================
(function init() {
    // 回车登录
    document.getElementById('login-password').addEventListener('keydown', function(e) {
        if (e.key === 'Enter') doLogin();
    });
    document.getElementById('login-username').addEventListener('keydown', function(e) {
        if (e.key === 'Enter') doLogin();
    });
    // 检查登录状态
    if (token) {
        api('/api/user').then(res => {
            if (res.code === 0) showMain();
            else doLogout();
        }).catch(() => {});
    }
})();