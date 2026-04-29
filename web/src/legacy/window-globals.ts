// Helling WebUI — typed access to legacy window.* UI bus.
//
// shell.jsx + infra.jsx + app.jsx attach toast / modal / nav helpers to
// window during initialization. Phase 2A page extractions need typed
// access; this module is the single bridge.
//
// Phase 2B replaces the entire surface with a useSyncExternalStore-
// backed UI store in `web/src/stores/ui-store.ts`. At that point every
// `import { ... } from '../../legacy/window-globals'` flips to
// `import { ... } from '../../stores/ui-store'` — same hook signatures.

export type ToastKind = 'info' | 'success' | 'warn' | 'danger';

export interface ToastBus {
  info?: (title: string, msg?: string) => void;
  success?: (title: string, msg?: string) => void;
  warn?: (title: string, msg?: string) => void;
  danger?: (title: string, msg?: string) => void;
  error?: (title: string, msg?: string) => void;
}

export interface ConfirmModalProps {
  title: string;
  body?: string;
  danger?: boolean;
  confirmLabel?: string;
  confirmMatch?: string;
  onConfirm?: () => void;
}

export type ModalKind = 'confirm' | 'create-vm' | 'install-app' | 'new-rule' | 'edit-cloud-init';

interface UIGlobals {
  toast?: ToastBus;
  openModal?: (kind: ModalKind, props?: Record<string, unknown>) => void;
  closeModal?: () => void;
  __nav?: (page: string) => void;
}

const ui = (): UIGlobals => (typeof window === 'undefined' ? {} : (window as unknown as UIGlobals));

export const toast = (kind: ToastKind, title: string, msg?: string): void => {
  const bus = ui().toast;
  if (!bus) return;
  const fn = bus[kind] ?? bus.info;
  fn?.(title, msg);
};

export const openConfirm = (props: ConfirmModalProps): void => {
  ui().openModal?.('confirm', props as unknown as Record<string, unknown>);
};

export const openModal = (kind: ModalKind, props?: Record<string, unknown>): void => {
  ui().openModal?.(kind, props);
};

export const closeModal = (): void => {
  ui().closeModal?.();
};

export const navigate = (page: string): void => {
  ui().__nav?.(page);
};
