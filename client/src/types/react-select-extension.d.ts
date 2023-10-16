import { CreatableProps, Props } from "react-select/creatable";

declare module "react-select/dist/declarations/src/Select" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
  }
  export interface CreatableProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
  }
}
