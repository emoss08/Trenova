/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FieldValues, useFormContext, useWatch } from "react-hook-form";

export const useFormValues = <T extends FieldValues>() => {
  const { getValues } = useFormContext<T>();

  return {
    ...useWatch(),
    ...getValues(),
  };
};
