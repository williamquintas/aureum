// Environment protection plugin
// Prevents accidental exposure of sensitive files

export const EnvProtection = async ({ project, client, $, directory, worktree }) => {
  return {
    "tool.execute.before": async (input, output) => {
      if (input.tool === "read") {
        const path = output.args.filePath || "";
        if (path.includes(".env") && !path.includes(".env.example") && !path.includes(".env.sample")) {
          throw new Error("Cannot read .env files (contains secrets). Use .env.example instead.");
        }
      }
    },
  }
}
