import '@emotion/react';
import { OttieTheme } from './types';

declare module '@emotion/react' {
  export interface Theme extends OttieTheme {}
}
