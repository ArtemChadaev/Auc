import {
  type RouteConfig,
  route,
  index,
} from "@react-router/dev/routes";

export default [
  route('/', './routes/titlePage/index.tsx', [
    index("./routes/titlePage/startPage.tsx"),
    route('list', "./routes/titlePage/listPage.tsx"),
    route('search', './routes/titlePage/searchPage.tsx'),
  ]),
  route("register", "./routes/register/index.tsx", [
    index("./routes/register/StepRegister.tsx"),
    route("confirm", "./routes/register/StepConfirm.tsx"),
    route("password", "./routes/register/StepPassword.tsx"),
    route("policies", "./routes/register/StepPolicies.tsx"),
  ]),
  // index("routes/.tsx"),
] satisfies RouteConfig;
