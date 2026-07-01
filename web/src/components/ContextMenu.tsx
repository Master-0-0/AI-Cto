import { useEffect, useRef, useState } from 'react';

export interface ContextMenuItem {
  label: string;
  icon?: string;
  onClick: () => void;
  disabled?: boolean;
  children?: ContextMenuItem[];
}

interface Props {
  x: number;
  y: number;
  items: ContextMenuItem[];
  onClose: () => void;
}

export default function ContextMenu({ x, y, items, onClose }: Props) {
  const menuRef = useRef<HTMLDivElement>(null);
  const submenuRef = useRef<HTMLDivElement>(null);
  const [openSubmenuIndex, setOpenSubmenuIndex] = useState<number | null>(null);
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Adjust position to keep menu within viewport
  const menuWidth = 180; // estimated
  const menuHeight = items.length * 36 + 8; // estimated row height * items + padding

  const adjustedX = Math.min(x, window.innerWidth - menuWidth - 8);
  const adjustedY = Math.min(y, window.innerHeight - menuHeight - 8);

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose();
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  // Cleanup close timeout on unmount
  useEffect(() => {
    return () => {
      if (closeTimeoutRef.current) clearTimeout(closeTimeoutRef.current);
    };
  }, []);

  // Close on click outside
  useEffect(() => {
    function handleMouseDown(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    }
    // Delay listener to avoid the same right-click event closing the menu
    const timer = setTimeout(() => {
      document.addEventListener('mousedown', handleMouseDown);
    }, 0);
    return () => {
      clearTimeout(timer);
      document.removeEventListener('mousedown', handleMouseDown);
    };
  }, [onClose]);

  function handleMouseEnter(index: number, item: ContextMenuItem) {
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
      closeTimeoutRef.current = null;
    }
    if (item.children && !item.disabled) {
      setOpenSubmenuIndex(index);
    } else {
      setOpenSubmenuIndex(null);
    }
  }

  function handleMouseLeave() {
    closeTimeoutRef.current = setTimeout(() => {
      setOpenSubmenuIndex(null);
    }, 150);
  }

  function handleSubmenuMouseEnter() {
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
      closeTimeoutRef.current = null;
    }
  }

  function handleSubmenuMouseLeave() {
    closeTimeoutRef.current = setTimeout(() => {
      setOpenSubmenuIndex(null);
    }, 150);
  }

  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{ left: adjustedX, top: adjustedY }}
    >
      {items.map((item, i) => (
        <div
          key={i}
          className={`context-menu-item${item.disabled ? ' disabled' : ''}`}
          onMouseEnter={() => handleMouseEnter(i, item)}
          onMouseLeave={handleMouseLeave}
          onClick={() => {
            if (item.disabled) return;
            if (item.children) return;
            item.onClick();
            onClose();
          }}
        >
          {item.icon && <span>{item.icon}</span>}
          <span>{item.label}</span>
          {item.children && <span className="context-menu-arrow">▶</span>}

          {item.children && openSubmenuIndex === i && (
            <div
              ref={submenuRef}
              className="context-menu context-menu-submenu"
              onMouseEnter={handleSubmenuMouseEnter}
              onMouseLeave={handleSubmenuMouseLeave}
            >
              {item.children.map((child, j) => (
                <div
                  key={j}
                  className={`context-menu-item${child.disabled ? ' disabled' : ''}`}
                  onClick={(e) => {
                    e.stopPropagation();
                    if (child.disabled) return;
                    child.onClick();
                    onClose();
                  }}
                >
                  {child.icon && <span>{child.icon}</span>}
                  <span>{child.label}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
