---
title: NetDispatch Improvements - Ports, Frontend, Documentation
date: 2026-03-31
status: approved
---

# NetDispatch Improvements Design

## Overview

This design covers 6 improvement tasks grouped into 3 phases:
1. Default ports configuration change
2. Frontend improvements (real-time updates, UI enhancements, responsive design)
3. Architecture documentation update

## Phase 1: Default Ports

### Change Summary
Update default proxy ports from 809/810 to 8009/8010.

### Files to Modify
- `pkg/config/config.go` - Update `DefaultConfig()` function
- `configs/config.yaml` - Update sample configuration

### Code Changes

```go
// pkg/config/config.go - DefaultConfig()
HTTP: PortConfig{
    Port:    8009,  // Changed from 809
    Enabled: true,
},
SOCKS5: SOCKSConfig{
    Port:    8010,  // Changed from 810
    Enabled: true,
},
```

### Impact
- New installations use 8009/8010 by default
- Existing configuration files are unaffected
- Documentation should reflect new defaults

---

## Phase 2: Frontend Improvements

### 2.1 Dashboard Real-Time Updates

**Current State**: Polling every 5 seconds via TanStack Query.

**New State**: WebSocket connection for true real-time updates.

**Implementation**:

1. Backend: Wire up existing WebSocket hub to broadcast traffic stats every 1-2 seconds
2. Frontend: Replace polling with WebSocket hook

```typescript
// web/src/pages/Dashboard.tsx
const useWebSocketStats = () => {
  const [stats, setStats] = useState<Stats | null>(null);

  useEffect(() => {
    const ws = new WebSocket(`ws://${window.location.host}/ws`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'traffic') {
        setStats(data.payload);
      }
    };

    ws.onclose = () => {
      // Reconnect logic
      setTimeout(() => window.location.reload(), 3000);
    };

    return () => ws.close();
  }, []);

  return stats;
};
```

### 2.2 Rules Page - Priority Explanation

**Problem**: Users don't understand if lower or higher priority is better.

**Solution**: Add visual hints in two places:

1. **Table Column Header** - Tooltip on hover
2. **Form Field** - Helper text below input

```typescript
// Table column with tooltip
{
  title: (
    <span>
      优先级
      <Tooltip title="数值越小优先级越高，规则越先匹配">
        <InfoCircleOutlined style={{ marginLeft: 4, color: '#999' }} />
      </Tooltip>
    </span>
  ),
  dataIndex: 'priority',
  key: 'priority',
  width: 80,
}

// Form field with helper text
<Form.Item
  label="优先级"
  name="priority"
  extra="数值越小优先级越高，范围 0-100"
>
  <InputNumber min={0} max={100} />
</Form.Item>
```

### 2.3 Help Page Updates

Update `web/src/pages/Help.tsx` with:

1. **Port Configuration Section**: Update references from 809/810 to 8009/8010
2. **Priority Explanation**: New section explaining rule priority
3. **Real-Time Traffic Section**: Document WebSocket-based monitoring
4. **General Cleanup**: Remove outdated content, improve clarity

### 2.4 Modern UI Enhancements

**Theme Improvements** (`web/src/index.css`):

```css
:root {
  --primary-color: #1890ff;
  --bg-color: #f0f2f5;
  --card-bg: #ffffff;
  --text-color: #333;
  --border-radius: 8px;
  --shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

/* Modern card styling */
.ant-card {
  border-radius: var(--border-radius);
  box-shadow: var(--shadow);
  transition: box-shadow 0.3s ease;
}

.ant-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}
```

### 2.5 Responsive Design

**Sidebar** (`web/src/components/Sidebar.tsx`):

- Desktop (>= 768px): Fixed 200px sidebar
- Mobile (< 768px): Hamburger menu triggers drawer

```typescript
const Sidebar: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const isMobile = useMediaQuery('(max-width: 768px)');

  if (isMobile) {
    return (
      <Drawer
        placement="left"
        visible={!collapsed}
        onClose={() => setCollapsed(true)}
      >
        <Menu items={menuItems} />
      </Drawer>
    );
  }

  return <Sider width={200}>...</Sider>;
};
```

**Layout Adjustments**:

- Dashboard stats cards: 4 columns → 2 columns (tablet) → 1 column (mobile)
- Tables: Horizontal scroll on small screens
- Forms: Full width on mobile

---

## Phase 3: Documentation Update

Update `docs/architecture.md`:

1. **Port References**: Change example configs to show 8009/8010
2. **WebSocket Section**: Document real-time update mechanism
3. **Frontend Section**: Add responsive design patterns
4. **Recent Features**: Add domain tree optimization, buffer pool
5. **Cleanup**: Remove outdated placeholder content

---

## File Summary

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Default ports 8009/8010 |
| `configs/config.yaml` | Sample config update |
| `web/src/pages/Dashboard.tsx` | WebSocket real-time |
| `web/src/pages/Rules.tsx` | Priority tooltips |
| `web/src/pages/Help.tsx` | Content updates |
| `web/src/index.css` | Modern theming |
| `web/src/components/Sidebar.tsx` | Responsive sidebar |
| `docs/architecture.md` | Documentation update |

---

## Success Criteria

1. New installations default to ports 8009/8010
2. Dashboard updates within 1-2 seconds (not 5)
3. Rules page clearly explains priority meaning
4. Help content is accurate and helpful
5. UI looks modern and works on mobile devices
6. Architecture docs reflect current state
