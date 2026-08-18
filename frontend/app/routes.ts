import {
  type RouteConfig,
  route,
  index,
} from "@react-router/dev/routes";

export default [
  route("register", "./routes/register/index.tsx", [
    index("./routes/register/StepRegister.tsx"),
    route("confirm", "./routes/register/StepConfirm.tsx"),
    route("password", "./routes/register/StepPassword.tsx"),
    route("policies", "./routes/register/StepPolicies.tsx"),
  ]),
  // index("routes/.tsx"),
] satisfies RouteConfig;
