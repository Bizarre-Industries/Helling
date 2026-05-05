// Vitest setup file. Registers @testing-library/jest-dom custom matchers
// (toBeInTheDocument, toHaveTextContent, etc.) on every test run.
import '@testing-library/jest-dom/vitest';

function getUsableStorage(storage: Storage | undefined): Storage | null {
  try {
    if (
      typeof storage?.getItem !== 'function' ||
      typeof storage?.setItem !== 'function' ||
      typeof storage?.removeItem !== 'function' ||
      typeof storage?.clear !== 'function'
    ) {
      return null;
    }

    const key = '__helling_storage_probe__';
    storage.setItem(key, '1');
    storage.removeItem(key);
    return storage;
  } catch {
    return null;
  }
}

function createMemoryStorage(): Storage {
  const values = new Map<string, string>();

  return {
    get length() {
      return values.size;
    },
    clear() {
      values.clear();
    },
    getItem(key: string) {
      return values.get(key) ?? null;
    },
    key(index: number) {
      return Array.from(values.keys())[index] ?? null;
    },
    removeItem(key: string) {
      values.delete(key);
    },
    setItem(key: string, value: string) {
      values.set(key, value);
    },
  };
}

// Node 25 exposes a global localStorage placeholder unless launched with a
// valid --localstorage-file. Vitest's jsdom window has the browser-compatible
// implementation the tests expect, so make that the global storage object.
if (typeof window !== 'undefined') {
  const localStorage = getUsableStorage(window.localStorage) ?? createMemoryStorage();
  const sessionStorage = getUsableStorage(window.sessionStorage) ?? createMemoryStorage();

  if (!getUsableStorage(globalThis.localStorage)) {
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: localStorage,
    });
  }

  if (!getUsableStorage(globalThis.sessionStorage)) {
    Object.defineProperty(globalThis, 'sessionStorage', {
      configurable: true,
      value: sessionStorage,
    });
  }
}
