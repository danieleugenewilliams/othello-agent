
At the end of this message, I will ask you to do something. Please follow the "Explore and Plan" workflow when you start.

Explore

First, use subagents sequentially to find and read all files, memories, and git commits that may be useful for implementing the ticket, either as examples or as edit targets. The subagents should build on the previous subagent input and return relevant file paths, and any other info that may be useful.

Plan

Next, think hard and write up a detailed implementation plan. Don't forget to include tests, lookbook components, and documentation. Use your judgement as to what is necessary, given the standards of this repo.

If there are things you are not sure about, use parallel subagents to do some web research. They should only return useful information, no noise.

DO NOT create technical debt. Build on the architecture and implementation foundation. If an approach is obsolete, recommend removing and replacing. But do not simply create new files without first verifying that a rearchitecture or refactor is a better option.

If there are things you still do not understand or questions you have for the user, pause here to ask them before continuing.

Write up your work

When you are happy with your work, write up a short report that could be used as the PR description. Include what you set out to do, the choices you made with their brief justification, and any commands you ran in the process that may be useful for future developers to know about.

$ARGUMENTS