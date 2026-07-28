import { lazy, Suspense, useEffect } from 'react'
import { BrowserRouter } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import './i18n';
import { useConfig } from './hooks';

const windowMap = {
  root: lazy(() => import('./window/ToolBar')),
  translate: lazy(() => import('./window/Translate')),
  screenshot: lazy(() => import('./window/Screenshot')),
};

function App({ variable }) {
  const { i18n } = useTranslation();
  const [appLanguage] = useConfig('app_language', 'zh_cn');
  const WindowComponent = windowMap[variable];

  useEffect(() => {
    if (appLanguage !== null) {
      i18n.changeLanguage(appLanguage);
    }
  }, [appLanguage])

  return (
    <BrowserRouter>
      <Suspense fallback={null}>
        <WindowComponent />
      </Suspense>
    </BrowserRouter>
  )
}

export default App
