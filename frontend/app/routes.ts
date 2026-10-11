import {
  type RouteConfig,
  route,
  index,
  layout,
} from "@react-router/dev/routes";

export default [
  layout("./components/layout/AppLayout.tsx", [
    index("./routes/home.tsx"),
    route("lots", "./routes/lots.tsx"),
    route("lots/new", "./routes/lots.new.tsx"),
    route("lot/:lotId", "./routes/lot.$lotId.tsx"),
    route("profile", "./routes/profile.tsx"),
    route("login", "./routes/login.tsx"),
    route("register", "./routes/register.tsx"),
    route("docs/:slug", "./routes/docs.$slug.tsx"),
  ]),
] satisfies RouteConfig;
