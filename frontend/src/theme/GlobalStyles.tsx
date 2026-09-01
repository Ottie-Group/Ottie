import { Global, css, useTheme } from '@emotion/react';

export function GlobalStyles() {
  const theme = useTheme();

  return (
    <Global
      styles={css`
        *,
        *::before,
        *::after {
          box-sizing: border-box;
          margin: 0;
          padding: 0;
        }

        body {
          background-color: ${theme.colors.bgPage};
          background-image: linear-gradient(${theme.colors.gridLine} 1.2px, transparent 1.2px),
            linear-gradient(90deg, ${theme.colors.gridLine} 1.2px, transparent 1.2px);
          background-size: 32px 32px;
          color: ${theme.colors.textDark};
          font-family: ${theme.fonts.body};
          min-height: 100vh;
          display: flex;
          flex-direction: column;
          align-items: center;
          padding: 32px 16px 80px;
          line-height: 1.5;
          -webkit-font-smoothing: antialiased;
          -moz-osx-font-smoothing: grayscale;
        }

        #root {
          width: 100%;
          display: flex;
          flex-direction: column;
          align-items: center;
        }

        a {
          color: inherit;
          text-decoration: none;
        }

        button,
        input,
        select,
        textarea {
          font-family: inherit;
        }
      `}
    />
  );
}
