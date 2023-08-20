import { Box, Card, Text } from "@mantine/core";
import React from "react";
import { CustomerRuleProfile } from "@/types/apps/customer";
import { useFormStyles } from "@/styles/FormStyles";
import { usePageStyles } from "@/styles/PageStyles";

type Props = {
  ruleProfile: CustomerRuleProfile;
};

export function CustomerRuleProfileForm({
  ruleProfile,
}: Props): React.ReactElement {
  const { classes } = useFormStyles();
  const { classes: pageClass } = usePageStyles();

  return (
    <Card className={pageClass.card} mt={20}>
      <Box>
        <Text className={classes.text} fw={600} fz={20}>
          Customer Rule Profile {ruleProfile.name}
        </Text>
      </Box>
    </Card>
  );
}
