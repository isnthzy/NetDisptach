import { Card, Typography, Collapse, Alert, Divider } from 'antd'

const { Paragraph, Text } = Typography

function Help() {
  return (
    <div>
      <Card title="使用手册">
        <Alert
          message="NetDispatch 使用指南"
          description="本手册将帮助您快速了解如何使用 NetDispatch 网络调度器。"
          type="info"
          showIcon
          style={{ marginBottom: 24 }}
        />

        <Collapse
          items={[
            {
              key: '1',
              label: '1. 什么是出口策略？',
              children: (
                <div>
                  <Paragraph>
                    <Text strong>出口策略</Text> 定义了代理请求如何出去到互联网。
                  </Paragraph>
                  <Paragraph>每个出口策略包含：</Paragraph>
                  <ul>
                    <li><Text strong>网卡</Text>：选择从哪个网络接口出去（如：网线、WiFi）</li>
                    <li><Text strong>代理服务器</Text>（可选）：是否通过上游代理转发请求</li>
                  </ul>
                  <Paragraph>
                    <Text type="secondary">示例：</Text>
                  </Paragraph>
                  <ul>
                    <li>「网线直连」：使用网线直接连接目标，不经过代理</li>
                    <li>「WiFi走代理」：使用 WiFi 网卡，并通过 SOCKS5 代理转发</li>
                  </ul>
                </div>
              ),
            },
            {
              key: '2',
              label: '2. 如何配置出口策略？',
              children: (
                <div>
                  <Paragraph>步骤：</Paragraph>
                  <ol>
                    <li>点击左侧菜单「出口策略」</li>
                    <li>点击「添加策略」按钮</li>
                    <li>输入策略名称（如：WiFi走代理）</li>
                    <li>选择要使用的网卡</li>
                    <li>如需使用代理，开启「使用代理服务器」开关
                      <ul>
                        <li>选择代理协议（SOCKS5 或 HTTP）</li>
                        <li>输入代理服务器地址和端口</li>
                        <li>如需认证，填写用户名和密码</li>
                      </ul>
                    </li>
                    <li>点击「确定」保存</li>
                  </ol>
                </div>
              ),
            },
            {
              key: '3',
              label: '3. 什么是路由规则？',
              children: (
                <div>
                  <Paragraph>
                    <Text strong>路由规则</Text> 决定了哪些请求使用哪个出口策略。
                  </Paragraph>
                  <Paragraph>规则按优先级从高到低匹配：</Paragraph>
                  <ul>
                    <li><Text strong>优先级</Text>：数值越小优先级越高，越先匹配</li>
                    <li>范围：0-100，建议留有一定间隔以便后续插入新规则</li>
                  </ul>
                  <Divider />
                  <Paragraph>匹配条件包括：</Paragraph>
                  <ul>
                    <li><Text strong>域名</Text>：支持通配符，如 *.google.com</li>
                    <li><Text strong>IP/CIDR</Text>：如 192.168.0.0/16、10.0.0.0/8</li>
                    <li><Text strong>端口</Text>：如 80、443、8080</li>
                  </ul>
                  <Paragraph>
                    <Text type="secondary">示例：</Text>
                  </Paragraph>
                  <ul>
                    <li>规则（优先级10）：*.google.com → WiFi走代理</li>
                    <li>规则（优先级20）：10.0.0.0/8 → 网线直连</li>
                    <li>规则（优先级100）：* → 默认出口（兜底规则）</li>
                  </ul>
                </div>
              ),
            },
            {
              key: '4',
              label: '4. 什么是黑白名单？',
              children: (
                <div>
                  <Paragraph>
                    路由规则支持三种名单类型：
                  </Paragraph>
                  <ul>
                    <li><Text strong>普通规则</Text>：按优先级匹配，匹配后执行指定动作</li>
                    <li><Text strong>白名单</Text>：仅允许匹配的地址通过，其他全部拒绝</li>
                    <li><Text strong>黑名单</Text>：阻止匹配的地址访问，其他全部允许</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>白名单使用场景：</Text></Paragraph>
                  <Paragraph>适用于只允许访问特定网站的场景，如企业内网只允许访问公司相关网站。</Paragraph>
                  <Paragraph>
                    <Text type="secondary">示例：白名单设置 *.company.com，则只有公司网站可访问，其他全部拒绝。</Text>
                  </Paragraph>
                  <Divider />
                  <Paragraph><Text strong>黑名单使用场景：</Text></Paragraph>
                  <Paragraph>适用于阻止特定网站或IP的场景，如阻止恶意网站、广告域名等。</Paragraph>
                  <Paragraph>
                    <Text type="secondary">示例：黑名单设置 *.badsite.com、192.168.100.0/24，则这些地址会被阻止。</Text>
                  </Paragraph>
                </div>
              ),
            },
            {
              key: '5',
              label: '5. 如何配置路由规则？',
              children: (
                <div>
                  <Paragraph>步骤：</Paragraph>
                  <ol>
                    <li>点击左侧菜单「路由规则」</li>
                    <li>点击「添加规则」按钮</li>
                    <li>输入规则名称</li>
                    <li>设置优先级（<Text type="warning">数值越小优先级越高</Text>，范围 0-100）</li>
                    <li>选择名单类型
                      <ul>
                        <li>普通规则：标准路由规则</li>
                        <li>白名单：仅允许匹配的地址</li>
                        <li>黑名单：阻止匹配的地址</li>
                      </ul>
                    </li>
                    <li>填写匹配条件
                      <ul>
                        <li>域名：每行一个，支持 *.example.com 格式</li>
                        <li>IP/CIDR：每行一个，如 192.168.0.0/16</li>
                        <li>端口：逗号分隔，如 80, 443</li>
                      </ul>
                    </li>
                    <li>选择动作
                      <ul>
                        <li>「转发到出口策略」：选择一个已创建的出口策略</li>
                        <li>「拒绝连接」：阻止该请求</li>
                      </ul>
                    </li>
                    <li>点击「确定」保存</li>
                  </ol>
                </div>
              ),
            },
            {
              key: '6',
              label: '6. 如何设置代理端口？',
              children: (
                <div>
                  <Paragraph>步骤：</Paragraph>
                  <ol>
                    <li>点击左侧菜单「设置」</li>
                    <li>在「代理端口」区域设置各协议端口
                      <ul>
                        <li>HTTP/HTTPS 代理端口（默认 8009）</li>
                        <li>SOCKS5 端口（默认 8010）</li>
                      </ul>
                    </li>
                    <li>可以单独启用/禁用各协议</li>
                    <li>点击「保存设置」</li>
                  </ol>
                </div>
              ),
            },
            {
              key: '7',
              label: '7. 如何设置绑定地址？',
              children: (
                <div>
                  <Paragraph>
                    <Text strong>绑定地址</Text> 决定了代理服务监听哪个网络接口。
                  </Paragraph>
                  <Paragraph><Text strong>自动选择规则</Text>（优先级从高到低）：</Paragraph>
                  <ul>
                    <li>WLAN（WiFi）的有效 IP</li>
                    <li>以太网（网线）的有效 IP</li>
                    <li>默认网关所在网卡</li>
                    <li>0.0.0.0（所有网卡）</li>
                  </ul>
                  <Paragraph>也可以手动选择具体的 IP 地址。</Paragraph>
                </div>
              ),
            },
            {
              key: '8',
              label: '8. 客户端如何配置代理？',
              children: (
                <div>
                  <Paragraph>浏览器或系统代理设置：</Paragraph>
                  <ul>
                    <li>HTTP/HTTPS 代理：服务器地址 + 端口 8009</li>
                    <li>SOCKS5 代理：服务器地址 + 端口 8010</li>
                  </ul>
                  <Paragraph>
                    <Text type="secondary">例如：服务器 IP 为 <YOUR_SERVER_IP>，HTTP 代理设为 <YOUR_SERVER_IP>:8009</Text>
                  </Paragraph>
                </div>
              ),
            },
            {
              key: '9',
              label: '9. 如何查看实时流量？',
              children: (
                <div>
                  <Paragraph>点击左侧菜单「仪表盘」可以查看：</Paragraph>
                  <ul>
                    <li>当前活跃连接数（实时更新）</li>
                    <li>入站/出站流量统计（实时更新）</li>
                    <li>实时流量图表（每2秒刷新）</li>
                    <li>最近连接列表</li>
                  </ul>
                  <Divider />
                  <Paragraph>
                    <Text type="secondary">页面顶部显示的 <Text code>实时连接</Text> 图标表示已连接 WebSocket，数据实时推送。</Text>
                  </Paragraph>
                </div>
              ),
            },
            {
              key: '10',
              label: '10. 系统托盘功能',
              children: (
                <div>
                  <Paragraph>
                    NetDispatch 启动后会最小化到系统托盘，方便后台运行。
                  </Paragraph>
                  <Paragraph><Text strong>托盘操作：</Text></Paragraph>
                  <ul>
                    <li><Text strong>左键双击</Text>：打开 Web 控制台</li>
                    <li><Text strong>右键菜单</Text>：
                      <ul>
                        <li>「打开网页」：打开 Web 控制台</li>
                        <li>「退出」：关闭程序</li>
                      </ul>
                    </li>
                  </ul>
                </div>
              ),
            },
            {
              key: '11',
              label: '11. Web 监控控制台是什么？',
              children: (
                <div>
                  <Paragraph>
                    <Text strong>Web 监控控制台</Text> 是 NetDispatch 提供的 Web 管理界面，通过浏览器访问。
                  </Paragraph>
                  <Paragraph><Text strong>功能包括：</Text></Paragraph>
                  <ul>
                    <li><Text strong>仪表盘</Text>：查看实时流量、活跃连接、流量图表</li>
                    <li><Text strong>出口策略</Text>：管理网卡和代理服务器组合</li>
                    <li><Text strong>路由规则</Text>：配置流量路由策略</li>
                    <li><Text strong>日志</Text>：实时查看连接日志</li>
                    <li><Text strong>设置</Text>：配置代理端口、绑定地址等</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>访问方式：</Text></Paragraph>
                  <ul>
                    <li>系统托盘右键 → 打开网页</li>
                    <li>系统托盘左键双击</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>监听设置：</Text></Paragraph>
                  <ul>
                    <li><Text code>127.0.0.1</Text>（默认）：仅本机可访问控制台（更安全）</li>
                    <li><Text code>0.0.0.0</Text>：监听所有网卡，局域网内其他设备可访问</li>
                  </ul>
                </div>
              ),
            },
            {
              key: '12',
              label: '12. 如何筛选不同网卡的流量日志？',
              children: (
                <div>
                  <Paragraph>
                    在「日志」页面，使用下拉菜单筛选不同出口策略的流量。
                  </Paragraph>
                  <Paragraph><Text strong>筛选选项：</Text></Paragraph>
                  <ul>
                    <li><Text strong>全部</Text>：显示所有日志</li>
                    <li><Text strong>网线</Text>：仅显示通过网线（以太网）转发的日志</li>
                    <li><Text strong>WiFi</Text>：仅显示通过 WiFi 转发的日志</li>
                    <li><Text strong>出口策略名</Text>：下拉列表会显示已配置的出口策略</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>使用场景：</Text></Paragraph>
                  <ul>
                    <li>排查某个网卡的网络问题</li>
                    <li>分析不同网络接口的流量分布</li>
                    <li>监控特定出口策略的使用情况</li>
                  </ul>
                </div>
              ),
            },
            {
              key: '13',
              label: '13. 常见问题',
              children: (
                <div>
                  <Paragraph><Text strong>Q: 代理无法连接？</Text></Paragraph>
                  <ul>
                    <li>检查防火墙是否允许程序监听端口</li>
                    <li>确认端口未被其他程序占用</li>
                    <li>检查绑定地址是否正确</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>Q: 出口策略不生效？</Text></Paragraph>
                  <ul>
                    <li>确认已创建出口策略</li>
                    <li>检查路由规则是否正确配置</li>
                    <li>确认规则的出口策略ID有效</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>Q: 网卡绑定失败？</Text></Paragraph>
                  <ul>
                    <li>确认网卡名称正确</li>
                    <li>检查网卡是否已启用</li>
                    <li>确认网卡有有效的IP地址</li>
                  </ul>
                  <Divider />
                  <Paragraph><Text strong>Q: 域名匹配不生效？</Text></Paragraph>
                  <ul>
                    <li><Text code>*.example.com</Text> 匹配 <Text code>www.example.com</Text> 等子域名</li>
                    <li><Text code>*.example.com</Text> <Text type="danger">不匹配</Text> <Text code>example.com</Text> 本身</li>
                    <li>如需匹配主域名，请同时添加 <Text code>example.com</Text> 和 <Text code>*.example.com</Text></li>
                  </ul>
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  )
}

export default Help
