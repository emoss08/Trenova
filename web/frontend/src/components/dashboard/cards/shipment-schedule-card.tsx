/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Timeline,
  TimelineContent,
  TimelineDot,
  TimelineHeading,
  TimelineItem,
  TimelineLine,
} from "@/components/ui/timeline";

export default function TimelineAlternate() {
  return (
    <Card className="relative col-span-4 lg:col-span-3">
      <CardContent className="relative p-0">
        <ScrollArea className="h-[40vh]">
          <Timeline positions="center">
            <TimelineItem status="done">
              <TimelineHeading side="left">Plan!</TimelineHeading>
              <TimelineHeading side="right" variant="secondary">
                Done (05/04/2024)
              </TimelineHeading>
              <TimelineDot status="done" />
              <TimelineLine done />
              <TimelineContent side="left">
                Before diving into coding, it is crucial to plan your software
                project thoroughly. This involves defining the project scope,
                setting clear objectives, and identifying the target audience.
                Additionally, creating a timeline and allocating resources
                appropriately will contribute to a successful development
                process.
              </TimelineContent>
            </TimelineItem>
            <TimelineItem status="done">
              <TimelineHeading side="right" className="text-destructive">
                Design
              </TimelineHeading>
              <TimelineHeading side="left" variant="secondary">
                Failed (05/04/2024)
              </TimelineHeading>
              <TimelineDot status="error" />
              <TimelineLine done />
              <TimelineContent>
                Designing your software involves creating a blueprint that
                outlines the structure, user interface, and functionality of
                your application. Consider user experience (UX) principles,
                wireframing, and prototyping to ensure an intuitive and visually
                appealing design.
              </TimelineContent>
            </TimelineItem>
            <TimelineItem status="done">
              <TimelineHeading side="left">Code</TimelineHeading>
              <TimelineHeading side="right" variant="secondary">
                Current step
              </TimelineHeading>
              <TimelineDot status="current" />
              <TimelineLine />
              <TimelineContent side="left">
                The coding phase involves translating your design into actual
                code. Choose a programming language and framework that aligns
                with your project requirements. Follow best practices, such as
                modular and reusable code, to enhance maintainability and
                scalability. Regularly test your code to identify and fix any
                bugs or errors.
              </TimelineContent>
            </TimelineItem>
            <TimelineItem>
              <TimelineHeading>Test</TimelineHeading>
              <TimelineHeading side="left" variant="secondary">
                Next step
              </TimelineHeading>
              <TimelineDot />
              <TimelineLine />
              <TimelineContent>
                Thorough testing is essential to ensure the quality and
                reliability of your software. Implement different testing
                methodologies, including unit testing, integration testing, and
                user acceptance testing. This helps identify and rectify any
                issues before deploying the software.
              </TimelineContent>
            </TimelineItem>
            <TimelineItem>
              <TimelineHeading side="left">Deploy</TimelineHeading>
              <TimelineHeading side="right" variant="secondary">
                Deadline (05/10/2024)
              </TimelineHeading>
              <TimelineDot />
              <TimelineLine />
              <TimelineContent side="left">
                Once your software has passed rigorous testing, it's time to
                deploy it. Consider the deployment environment, whether it's
                on-premises or in the cloud. Ensure proper documentation and
                provide clear instructions for installation and configuration.
              </TimelineContent>
            </TimelineItem>
            <TimelineItem>
              <TimelineDot />
              <TimelineHeading>Done!</TimelineHeading>
              <TimelineHeading side="left" variant="secondary">
                Here everything ends
              </TimelineHeading>
            </TimelineItem>
          </Timeline>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
