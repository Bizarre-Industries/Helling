import { Suspense, lazy } from 'react';

import openapi from '../../../../api/openapi.yaml?raw';

const ScalarReference = lazy(async () => {
  const [mod, style] = await Promise.all([
    import('@scalar/api-reference-react'),
    import('@scalar/api-reference-react/style.css?url'),
  ]);
  ensureScalarStylesheet(style.default);
  return {
    default: function ScalarReferenceInner() {
      return (
        <mod.ApiReferenceReact
          configuration={{
            content: openapi,
            hideDownloadButton: false,
            hideModels: false,
            layout: 'modern',
            theme: 'kepler',
          }}
        />
      );
    },
  };
});

function ensureScalarStylesheet(href: string) {
  const id = 'scalar-api-reference-css';
  if (document.getElementById(id)) return;
  const link = document.createElement('link');
  link.id = id;
  link.rel = 'stylesheet';
  link.href = href;
  document.head.appendChild(link);
}

export default function PageAPIDocs() {
  return (
    <div style={{ minHeight: 'calc(100vh - 96px)', background: 'var(--h-bg)' }}>
      <Suspense fallback={<div style={{ padding: 24 }}>Loading API reference...</div>}>
        <ScalarReference />
      </Suspense>
    </div>
  );
}
