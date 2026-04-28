// Helling WebUI — Users page (extracted from pages.jsx during Phase 2A).
//
// Reads users from the legacy mocks shim. Phase 3A replaces with a real
// users-query hook fed by `/api/v1/users`; this file's import flips, body
// stays.

import { getUsers } from '../../legacy/mocks';
import { I } from '../../primitives/icon';

export default function PageUsers() {
  const users = getUsers();
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button type="button" className="on">
              Users{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {users.length}
              </span>
            </button>
            <button type="button">
              Tokens{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                6
              </span>
            </button>
            <button type="button">
              Roles{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                3
              </span>
            </button>
            <button type="button">SSH keys</button>
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm">
            <I n="key-round" s={13} /> Create API token
          </button>
          <button type="button" className="btn btn--sm btn--primary">
            <I n="user-plus" s={13} /> Invite user
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th>User</th>
            <th>Role</th>
            <th>2FA</th>
            <th>Last login</th>
            <th>Sessions</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.name}>
              <td>
                <span className="mono" style={{ fontWeight: 600 }}>
                  {u.name}
                </span>{' '}
                <span className="dim">· {u.name}@helling.local</span>
              </td>
              <td>
                <span
                  className="badge mono"
                  style={{ color: u.role === 'admin' ? 'var(--h-accent)' : 'var(--h-text-2)' }}
                >
                  {u.role}
                </span>
              </td>
              <td>
                {u.twofa ? (
                  <span style={{ color: 'var(--h-success)' }}>✓ enabled</span>
                ) : (
                  <span style={{ color: 'var(--h-warn)' }}>! disabled</span>
                )}
              </td>
              <td className="mono dim">{u.lastLogin}</td>
              <td className="mono">1</td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                <button type="button" className="btn btn--sm btn--ghost">
                  Impersonate
                </button>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="pencil" s={13} />
                </button>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="trash-2" s={13} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
